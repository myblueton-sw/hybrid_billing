import multiprocessing
import os
import sys
import tempfile
import threading
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from modules.focus_import import file_lock


def contender(path, ready, release, legacy=False):
    if legacy:
        import fcntl
        with open(path, 'a') as stream:
            fcntl.flock(stream, fcntl.LOCK_EX)
            ready.send('acquired')
            release.wait(8)
            fcntl.flock(stream, fcntl.LOCK_UN)
    else:
        with file_lock.exclusive_lock(path):
            ready.send('acquired')
            release.wait(8)
    ready.close()


class NativeLockTests(unittest.TestCase):
    def test_process_exclusion_and_legacy_compatibility(self):
        context = multiprocessing.get_context('spawn')
        for first_legacy, second_legacy in [(False, False), (True, False), (False, True)]:
            if os.name != 'posix' and (first_legacy or second_legacy):
                continue
            with tempfile.TemporaryDirectory() as folder:
                path = str(Path(folder) / 'coordinator.lock')
                r1, w1 = context.Pipe(False)
                r2, w2 = context.Pipe(False)
                exit1, exit2 = context.Event(), context.Event()
                first = context.Process(target=contender, args=(path, w1, exit1, first_legacy))
                second = context.Process(target=contender, args=(path, w2, exit2, second_legacy))
                try:
                    first.start()
                    self.assertTrue(r1.poll(5))
                    self.assertEqual(r1.recv(), 'acquired')
                    second.start()
                    self.assertFalse(r2.poll(.15))
                    exit1.set()
                    self.assertTrue(r2.poll(5))
                    self.assertEqual(r2.recv(), 'acquired')
                    exit2.set()
                    first.join(5)
                    second.join(5)
                    self.assertEqual(first.exitcode, 0)
                    self.assertEqual(second.exitcode, 0)
                    self.assertTrue(Path(path).is_file())
                    self.assertEqual(Path(path).read_bytes(), b'')
                finally:
                    exit1.set()
                    exit2.set()
                    for child in [first, second]:
                        if child.pid:
                            child.join(3)
                            if child.is_alive():
                                child.terminate()
                                child.join(3)
                    for connection in [r1, w1, r2, w2]:
                        connection.close()

    def test_independent_threads_and_body_exception_release(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'lock'
            entered = threading.Event()
            def waiting():
                with file_lock.exclusive_lock(path):
                    entered.set()
            with file_lock.exclusive_lock(path):
                thread = threading.Thread(target=waiting)
                thread.start()
                self.assertFalse(entered.wait(.1))
            self.assertTrue(entered.wait(5))
            thread.join(5)
            self.assertFalse(thread.is_alive())
            with self.assertRaisesRegex(ValueError, 'body failure'):
                with file_lock.exclusive_lock(path):
                    raise ValueError('body failure')
            with file_lock.exclusive_lock(path):
                self.assertTrue(path.exists())


class AdapterContractTests(unittest.TestCase):
    def run_windows(self, callback, body):
        fake = SimpleNamespace(LK_LOCK=1, LK_UNLCK=2, locking=callback)
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'lock'
            with patch.object(file_lock, 'os', SimpleNamespace(name='nt')), patch.dict(sys.modules, {'msvcrt': fake}):
                body(path)
            self.assertEqual(path.read_bytes(), b'')

    def test_windows_offset_length_release_without_truncation(self):
        calls = []
        def native(fd, mode, length):
            calls.append((fd, mode, length, os.lseek(fd, 0, os.SEEK_CUR)))
            os.lseek(fd, 7, os.SEEK_SET)
        def body(path):
            with file_lock.exclusive_lock(path):
                self.assertEqual(len(calls), 1)
        self.run_windows(native, body)
        self.assertEqual([(m, n, pos) for _, m, n, pos in calls], [(1, 1, 0), (2, 1, 0)])
        for fd, *_ in calls:
            with self.assertRaises(OSError):
                os.fstat(fd)

    def test_windows_acquisition_failure_never_enters_or_unlocks(self):
        calls = []
        def native(fd, mode, length):
            calls.append((fd, mode))
            raise OSError('contention')
        def body(path):
            with self.assertRaisesRegex(OSError, 'contention'):
                with file_lock.exclusive_lock(path):
                    self.fail('unlocked entry')
        self.run_windows(native, body)
        self.assertEqual([mode for _, mode in calls], [1])
        with self.assertRaises(OSError):
            os.fstat(calls[0][0])

    def test_windows_body_and_unlock_failure_still_close_handle(self):
        for unlock_fails in [False, True]:
            calls = []
            def native(fd, mode, length):
                calls.append((fd, mode))
                if mode == 2 and unlock_fails:
                    raise OSError('unlock failure')
            def body(path):
                with self.assertRaises(OSError if unlock_fails else ValueError):
                    with file_lock.exclusive_lock(path):
                        raise ValueError('body failure')
            self.run_windows(native, body)
            self.assertEqual([mode for _, mode in calls], [1, 2])
            with self.assertRaises(OSError):
                os.fstat(calls[0][0])

    def test_unknown_platform_fails_closed(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / 'lock'
            with patch.object(file_lock, 'os', SimpleNamespace(name='unknown')):
                with self.assertRaisesRegex(RuntimeError, 'lock_platform_unsupported'):
                    with file_lock.exclusive_lock(path):
                        self.fail('unlocked entry')


if __name__ == '__main__':
    unittest.main()
