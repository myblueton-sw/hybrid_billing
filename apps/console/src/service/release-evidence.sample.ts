export const sample = {
  "revision": "r21",
  "now": "2026-11-17T18:09:16Z",
  "state": "configured",
  "candidate": {"version": "sample-candidate-1", "build": "0.0.2", "release": "not_connected"},
  "matrix": {
    "origin": "sample_seed",
    "paths": {
      "total": 2,
      "more": false,
      "rows": [
        {
          "id": "sample-path-api",
          "provider": "sample-provider",
          "contract": "sample-contract",
          "role": "sample-role",
          "version": "sample-1",
          "issuance": "api",
          "template": null,
          "state": "designed",
          "build": null,
          "requirements": {
            "total": 3,
            "more": false,
            "rows": [
              {"label": "sample/sample-approval", "target": "trace-requirement-2", "state": "tested"},
              {"label": "sample/sample-common-xlsx", "target": "trace-requirement-1", "state": "tested"},
              {"label": "sample/sample-recovery", "target": "trace-requirement-3", "state": "failing"}
            ]
          },
          "columns": {
            "design": {"count": 3, "of": 3},
            "implementation": {"count": 3, "of": 3},
            "test": {"count": 2, "of": 3},
            "customer_proof": {"count": 0, "of": 3},
            "failing": 1
          },
          "categories": {"amount": true, "approval": true, "bulk": false, "duplicate": true, "isolation": true, "recovery": true},
          "reconciliations": {"total": 1, "more": false, "rows": [{"period": "2026-10", "matched": true, "differences": 0}]},
          "last_observed": "2026-11-17T18:09:16Z",
          "blockers": [
            {"problem": "blocking", "subject": "sample/sample-recovery"},
            {"problem": "not_tested", "subject": "sample/sample-recovery"},
            {"problem": "path_evidence", "subject": "sample-path-api"},
            {"problem": "path_inactive", "subject": "sample-path-api"}
          ],
          "activation": "not_connected",
          "support_decision": "not_connected",
          "customer_proof_intake": "not_connected"
        },
        {
          "id": "sample-path-xlsx",
          "provider": "sample-provider",
          "contract": "sample-contract",
          "role": "sample-role",
          "version": "sample-1",
          "issuance": "xlsx",
          "template": "sample-template-xlsx",
          "state": "designed",
          "build": null,
          "requirements": {
            "total": 2,
            "more": false,
            "rows": [
              {"label": "sample/sample-common-xlsx", "target": "trace-requirement-1", "state": "tested"},
              {"label": "sample/sample-erp-template", "target": "trace-requirement-4", "state": "designed"}
            ]
          },
          "columns": {
            "design": {"count": 2, "of": 2},
            "implementation": {"count": 1, "of": 2},
            "test": {"count": 1, "of": 2},
            "customer_proof": {"count": 0, "of": 2},
            "failing": 0
          },
          "categories": {"amount": true, "approval": false, "bulk": false, "duplicate": false, "isolation": true, "recovery": false},
          "reconciliations": {"total": 1, "more": false, "rows": [{"period": "2026-10", "matched": false, "differences": 3}]},
          "last_observed": "2026-11-17T18:09:16Z",
          "blockers": [
            {"problem": "not_tested", "subject": "sample/sample-erp-template"},
            {"problem": "path_evidence", "subject": "sample-path-xlsx"},
            {"problem": "path_inactive", "subject": "sample-path-xlsx"},
            {"problem": "path_reconciliation", "subject": "sample-path-xlsx"}
          ],
          "activation": "not_connected",
          "support_decision": "not_connected",
          "customer_proof_intake": "not_connected"
        }
      ]
    }
  },
  "requirements": {
    "origin": "sample_seed",
    "requirements": {
      "total": 6,
      "more": false,
      "rows": [
        {
          "target": "trace-requirement-1",
          "label": "sample/sample-common-xlsx",
          "namespace": "sample",
          "id": "sample-common-xlsx",
          "source": "sample-source",
          "owner": "sample-owner",
          "interfaces": ["sample-api"],
          "common": true,
          "blocking": false,
          "design": "f9b21c43ae5cf6d202f64ed42a6f792d3d71a4fa98635da023b88cf387fae55d",
          "commit": "00624a3c1c736f9120dbef6803bc256996fd2a9c",
          "retired": false,
          "replacement": null,
          "states": {"0.0.1": "implemented", "0.0.2": "tested"},
          "builds_more": false,
          "tests": {
            "total": 2,
            "more": false,
            "rows": [
              {
                "id": "t-xlsx-amount",
                "category": "amount",
                "counted": true,
                "latest_outcome": "pass",
                "passes": 1,
                "failures": 0,
                "not_run_reason": null,
                "not_counted_reason": null
              },
              {
                "id": "t-xlsx-isolation",
                "category": "isolation",
                "counted": true,
                "latest_outcome": "pass",
                "passes": 1,
                "failures": 0,
                "not_run_reason": null,
                "not_counted_reason": null
              }
            ]
          },
          "retire": null,
          "replacements": {"total": 0, "more": false, "rows": []},
          "replacement_required": true,
          "can_retire": false,
          "retire_blocked": "no_replacement"
        },
        {
          "target": "trace-requirement-2",
          "label": "sample/sample-approval",
          "namespace": "sample",
          "id": "sample-approval",
          "source": "sample-source",
          "owner": "sample-owner",
          "interfaces": ["sample-api"],
          "common": false,
          "blocking": false,
          "design": "c7481a9feb2ea8c38cea5569791708293c436dee38205bcd5147f4cef14f52e2",
          "commit": "00624a3c1c736f9120dbef6803bc256996fd2a9c",
          "retired": false,
          "replacement": null,
          "states": {"0.0.1": "implemented", "0.0.2": "tested"},
          "builds_more": false,
          "tests": {
            "total": 2,
            "more": false,
            "rows": [
              {
                "id": "t-approval",
                "category": "approval",
                "counted": true,
                "latest_outcome": "pass",
                "passes": 1,
                "failures": 1,
                "not_run_reason": null,
                "not_counted_reason": null
              },
              {
                "id": "t-duplicate",
                "category": "duplicate",
                "counted": true,
                "latest_outcome": "pass",
                "passes": 1,
                "failures": 0,
                "not_run_reason": null,
                "not_counted_reason": null
              }
            ]
          },
          "retire": null,
          "replacements": {
            "total": 4,
            "more": false,
            "rows": [
              {"target": "trace-requirement-3", "label": "sample/sample-recovery"},
              {"target": "trace-requirement-4", "label": "sample/sample-erp-template"},
              {"target": "trace-requirement-5", "label": "sample/sample-api-export"},
              {"target": "trace-requirement-6", "label": "sample/sample-legacy-report"}
            ]
          },
          "replacement_required": false,
          "can_retire": true,
          "retire_blocked": ""
        },
        {
          "target": "trace-requirement-3",
          "label": "sample/sample-recovery",
          "namespace": "sample",
          "id": "sample-recovery",
          "source": "sample-source",
          "owner": "sample-owner",
          "interfaces": ["sample-api"],
          "common": false,
          "blocking": true,
          "design": "f4e98eef4adad6691049e20ff4047e42fcb04b29291d6cbb90feaf89803ef29a",
          "commit": "00624a3c1c736f9120dbef6803bc256996fd2a9c",
          "retired": false,
          "replacement": null,
          "states": {"0.0.1": "implemented", "0.0.2": "failing"},
          "builds_more": false,
          "tests": {
            "total": 2,
            "more": false,
            "rows": [
              {
                "id": "t-bulk",
                "category": "bulk",
                "counted": true,
                "latest_outcome": "fail",
                "passes": 0,
                "failures": 1,
                "not_run_reason": null,
                "not_counted_reason": null
              },
              {
                "id": "t-recovery",
                "category": "recovery",
                "counted": true,
                "latest_outcome": "pass",
                "passes": 1,
                "failures": 0,
                "not_run_reason": null,
                "not_counted_reason": null
              }
            ]
          },
          "retire": null,
          "replacements": {"total": 0, "more": false, "rows": []},
          "replacement_required": false,
          "can_retire": true,
          "retire_blocked": ""
        },
        {
          "target": "trace-requirement-4",
          "label": "sample/sample-erp-template",
          "namespace": "sample",
          "id": "sample-erp-template",
          "source": "sample-source",
          "owner": "sample-owner",
          "interfaces": ["sample-api"],
          "common": false,
          "blocking": false,
          "design": "ed3b592ed70a2c0d2e9d92ec4f317d8a601f1b4180210e6c3520739ec63c86f6",
          "commit": null,
          "retired": false,
          "replacement": null,
          "states": {"0.0.1": "designed", "0.0.2": "designed"},
          "builds_more": false,
          "tests": {
            "total": 1,
            "more": false,
            "rows": [
              {
                "id": "t-erp-template",
                "category": null,
                "counted": false,
                "latest_outcome": null,
                "passes": 0,
                "failures": 0,
                "not_run_reason": "no_result_recorded",
                "not_counted_reason": null
              }
            ]
          },
          "retire": null,
          "replacements": {
            "total": 2,
            "more": false,
            "rows": [{"target": "trace-requirement-5", "label": "sample/sample-api-export"}, {"target": "trace-requirement-6", "label": "sample/sample-legacy-report"}]
          },
          "replacement_required": false,
          "can_retire": true,
          "retire_blocked": ""
        },
        {
          "target": "trace-requirement-5",
          "label": "sample/sample-api-export",
          "namespace": "sample",
          "id": "sample-api-export",
          "source": "sample-source",
          "owner": "sample-owner",
          "interfaces": ["sample-api"],
          "common": false,
          "blocking": false,
          "design": "83eb0ea55f99d14beedf8c67ac50d5ae2bb88b11476711ef0ff93d17cdc41efa",
          "commit": "00624a3c1c736f9120dbef6803bc256996fd2a9c",
          "retired": false,
          "replacement": null,
          "states": {"0.0.1": "implemented", "0.0.2": "implemented"},
          "builds_more": false,
          "tests": {
            "total": 1,
            "more": false,
            "rows": [
              {
                "id": "t-api-export",
                "category": null,
                "counted": false,
                "latest_outcome": null,
                "passes": 0,
                "failures": 0,
                "not_run_reason": "no_result_recorded",
                "not_counted_reason": null
              }
            ]
          },
          "retire": null,
          "replacements": {"total": 1, "more": false, "rows": [{"target": "trace-requirement-6", "label": "sample/sample-legacy-report"}]},
          "replacement_required": false,
          "can_retire": true,
          "retire_blocked": ""
        },
        {
          "target": "trace-requirement-6",
          "label": "sample/sample-legacy-report",
          "namespace": "sample",
          "id": "sample-legacy-report",
          "source": "sample-source",
          "owner": "sample-owner",
          "interfaces": ["sample-api"],
          "common": false,
          "blocking": false,
          "design": "861149244559aaf400898a6cc64ced7bcc3b1bb22e7978d564e55c26215c1341",
          "commit": "00624a3c1c736f9120dbef6803bc256996fd2a9c",
          "retired": false,
          "replacement": null,
          "states": {"0.0.1": "implemented", "0.0.2": "implemented"},
          "builds_more": false,
          "tests": {
            "total": 1,
            "more": false,
            "rows": [
              {
                "id": "t-legacy",
                "category": null,
                "counted": false,
                "latest_outcome": null,
                "passes": 0,
                "failures": 0,
                "not_run_reason": "no_result_recorded",
                "not_counted_reason": null
              }
            ]
          },
          "retire": 19,
          "replacements": {"total": 0, "more": false, "rows": []},
          "replacement_required": false,
          "can_retire": false,
          "retire_blocked": "already_proposed"
        }
      ]
    },
    "gate": {
      "ready": false,
      "problems": {
        "total": 5,
        "more": false,
        "rows": [
          {"problem": "blocking", "subject": "sample/sample-recovery"},
          {"problem": "not_tested", "subject": "sample/sample-api-export"},
          {"problem": "not_tested", "subject": "sample/sample-recovery"},
          {"problem": "path_evidence", "subject": "sample-path-api"},
          {"problem": "path_inactive", "subject": "sample-path-api"}
        ]
      },
      "retired_blocking": [],
      "templates": [],
      "release": "not_connected"
    }
  },
  "artifacts": {
    "origin": "sample_seed",
    "builds": {
      "total": 2,
      "more": false,
      "rows": [
        {"build": "0.0.2", "commit": "ed56c543e6b2805b470137ec4d5b309eebf0b446", "order": 2},
        {"build": "0.0.1", "commit": "c91ea022e917a5d43b1694acedce175d857d6b2b", "order": 1}
      ]
    },
    "harnesses": {
      "total": 2,
      "more": false,
      "rows": [
        {"module": "release_trace", "id": "sample-trace-harness", "revocation": "not_connected"},
        {"module": "perf_gate", "id": "sample-perf-harness", "revocation": "not_connected"}
      ]
    },
    "results": {
      "total": 2,
      "more": false,
      "rows": [
        {
          "namespace": "sample",
          "test": "t-xlsx-amount",
          "build": "0.0.1",
          "counted": false,
          "latest_outcome": "pass",
          "evidence": "227f758983422c26f1b9244199b2238bd8639a5ae2abe8d3b53a0e1bd4f264b2",
          "harness": "sample-trace-harness",
          "at": "2026-11-17T18:09:16Z",
          "passes": 1,
          "failures": 0,
          "not_counted_reason": "build_predates_implementation"
        },
        {
          "namespace": "sample",
          "test": "t-bulk",
          "build": "0.0.2",
          "counted": true,
          "latest_outcome": "fail",
          "evidence": "3ceaab85f8ef942b7edd33050190f0fea7c821bdbf5ece308f98a7f5af31fbe2",
          "harness": "sample-trace-harness",
          "at": "2026-11-17T18:09:16Z",
          "passes": 0,
          "failures": 1,
          "not_counted_reason": null
        }
      ]
    },
    "files": "not_connected",
    "logs": "not_connected",
    "environment": "not_connected",
    "content": "not_connected"
  },
  "load": {
    "origin": "sample_seed",
    "hardware": {
      "total": 1,
      "more": false,
      "rows": [
        {
          "id": "sample-hw-small",
          "cpus": 8,
          "memory_mib": 32768,
          "disk_gib": 500,
          "storage": "ssd",
          "spec": "5b043974a366767c6b2482fb3384f94d588fa390b2259bdb1b1896fc2102b6d0"
        }
      ]
    },
    "profiles": {
      "total": 4,
      "more": false,
      "rows": [
        {
          "target": "perf-profile:sample-e2e",
          "id": "sample-e2e",
          "gate": "e2e",
          "axes": {
            "rows_run": 100000,
            "rows_day": 0,
            "rows_month": 0,
            "customers": 0,
            "accounts": 0,
            "shared_pools": 0,
            "skew_permille": 0,
            "fan_out": 0,
            "concurrent_closes": 0,
            "peak_window_minutes": 0,
            "retention_months": 0,
            "failure_injections": 0,
            "backlog_rows": 0
          },
          "dataset": "37fe04d5657b49632b454411404565fae5e911051176f76138d2cd08cf4f8458",
          "expected": "7b33771e5ef5364ddeb7b98911d631ed63cead5bbf58ab6db39b0ee1e5e87485",
          "retired": false,
          "active_budget": {
            "id": "sample-budget-e2e",
            "state": "proposed",
            "hardware": "sample-hw-small",
            "limits": {"completion_ms": 600000, "max_rss_bytes": 4294967296, "p95_ms": 500, "p99_ms": 1000, "throughput_rows_s": 200},
            "proposer": {"id": "sample-admin", "login": "admin@example.invalid", "directory": true},
            "approver": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
            "approval_by": null,
            "approved_by": null,
            "approved_at": null
          },
          "status": "provisional",
          "runs": {
            "total": 1,
            "more": false,
            "rows": [
              {
                "number": 1,
                "build": "0.0.2",
                "hardware": "sample-hw-small",
                "rows": 100000,
                "duration_s": 420,
                "mismatches": 0,
                "interrupted": false,
                "metrics": {
                  "completion_ms": 420000,
                  "db_io_bytes": 1073741824,
                  "lock_wait_ms": 40,
                  "max_rss_bytes": 2147483648,
                  "p95_ms": 300,
                  "p99_ms": 800,
                  "resume_ms": null,
                  "throughput_rows_s": 240,
                  "wal_bytes": 536870912
                },
                "harness": "sample-perf-harness",
                "at": "2026-11-17T18:09:16Z",
                "verdict": "provisional"
              }
            ]
          },
          "retire": null,
          "can_retire": true,
          "retire_blocked": ""
        },
        {
          "target": "perf-profile:sample-daily",
          "id": "sample-daily",
          "gate": "daily_inflow",
          "axes": {
            "rows_run": 0,
            "rows_day": 1200000,
            "rows_month": 0,
            "customers": 0,
            "accounts": 0,
            "shared_pools": 0,
            "skew_permille": 0,
            "fan_out": 0,
            "concurrent_closes": 0,
            "peak_window_minutes": 0,
            "retention_months": 0,
            "failure_injections": 0,
            "backlog_rows": 0
          },
          "dataset": "895d62e97bb59e5ef8fc223299f81ff416fdeb1f63904c3a4d8d79beb17611cf",
          "expected": "5cd8c03f435e68659c185fc1a1b6b914b0e15a5ed02d4b27925dca7820d29924",
          "retired": false,
          "active_budget": {
            "id": "sample-budget-daily",
            "state": "proposed",
            "hardware": "sample-hw-small",
            "limits": {"completion_ms": 86400000, "max_rss_bytes": 4294967296, "p95_ms": 500, "p99_ms": 1000, "throughput_rows_s": 20},
            "proposer": {"id": "sample-admin", "login": "admin@example.invalid", "directory": true},
            "approver": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
            "approval_by": null,
            "approved_by": null,
            "approved_at": null
          },
          "status": "over_budget",
          "runs": {
            "total": 1,
            "more": false,
            "rows": [
              {
                "number": 2,
                "build": "0.0.2",
                "hardware": "sample-hw-small",
                "rows": 1200000,
                "duration_s": 86400,
                "mismatches": 0,
                "interrupted": false,
                "metrics": {
                  "completion_ms": 86400000,
                  "db_io_bytes": 1073741824,
                  "lock_wait_ms": 40,
                  "max_rss_bytes": 3221225472,
                  "p95_ms": 400,
                  "p99_ms": 1500,
                  "resume_ms": null,
                  "throughput_rows_s": 13,
                  "wal_bytes": 536870912
                },
                "harness": "sample-perf-harness",
                "at": "2026-11-17T18:09:16Z",
                "verdict": "over_budget"
              }
            ]
          },
          "retire": null,
          "can_retire": true,
          "retire_blocked": ""
        },
        {
          "target": "perf-profile:sample-month",
          "id": "sample-month",
          "gate": "monthly_close",
          "axes": {
            "rows_run": 0,
            "rows_day": 0,
            "rows_month": 30000000,
            "customers": 0,
            "accounts": 0,
            "shared_pools": 0,
            "skew_permille": 0,
            "fan_out": 0,
            "concurrent_closes": 0,
            "peak_window_minutes": 0,
            "retention_months": 0,
            "failure_injections": 0,
            "backlog_rows": 0
          },
          "dataset": "30edf4e86ed5c2912b1aa01f0e620aad9bd90bd62ab0a755181b79a0dc27ac34",
          "expected": "af4a5545592bbce45f253f4a353534d796454713d2015aa3ae34154fefd35a7b",
          "retired": false,
          "active_budget": null,
          "status": "incomplete",
          "runs": {
            "total": 1,
            "more": false,
            "rows": [
              {
                "number": 3,
                "build": "0.0.2",
                "hardware": "sample-hw-small",
                "rows": 1000000,
                "duration_s": 3600,
                "mismatches": 0,
                "interrupted": false,
                "metrics": {
                  "completion_ms": 3600000,
                  "db_io_bytes": 1073741824,
                  "lock_wait_ms": 40,
                  "max_rss_bytes": 3221225472,
                  "p95_ms": 400,
                  "p99_ms": 900,
                  "resume_ms": null,
                  "throughput_rows_s": 278,
                  "wal_bytes": 536870912
                },
                "harness": "sample-perf-harness",
                "at": "2026-11-17T18:09:16Z",
                "verdict": "incomplete"
              }
            ]
          },
          "retire": 9,
          "can_retire": false,
          "retire_blocked": "already_proposed"
        },
        {
          "target": "perf-profile:sample-resume",
          "id": "sample-resume",
          "gate": "interrupt_resume",
          "axes": {
            "rows_run": 100000,
            "rows_day": 0,
            "rows_month": 0,
            "customers": 0,
            "accounts": 0,
            "shared_pools": 0,
            "skew_permille": 0,
            "fan_out": 0,
            "concurrent_closes": 0,
            "peak_window_minutes": 0,
            "retention_months": 0,
            "failure_injections": 1,
            "backlog_rows": 0
          },
          "dataset": "e42ceed922976894241ea9a0811720d45a5350358ff55b45bd793efcf3a11064",
          "expected": "ee842a0ad2897479e74b96732fe87e8ad238aaa85bdfc81adbec27bbc83567b4",
          "retired": false,
          "active_budget": null,
          "status": "not_run",
          "runs": {"total": 0, "more": false, "rows": []},
          "retire": null,
          "can_retire": true,
          "retire_blocked": ""
        }
      ]
    },
    "readiness": {
      "ready": false,
      "gates": [
        {"gate": "e2e", "status": "provisional", "profiles": [{"profile": "sample-e2e", "status": "provisional", "run": 1, "failures": 0}], "retired": []},
        {"gate": "daily_inflow", "status": "failing", "profiles": [{"profile": "sample-daily", "status": "over_budget", "run": 2, "failures": 0}], "retired": []},
        {"gate": "monthly_close", "status": "failing", "profiles": [{"profile": "sample-month", "status": "incomplete", "run": 3, "failures": 0}], "retired": []},
        {"gate": "interrupt_resume", "status": "not_run", "profiles": [{"profile": "sample-resume", "status": "not_run", "run": 0, "failures": 0}], "retired": []},
        {"gate": "catch_up", "status": "not_run", "profiles": [], "retired": []}
      ]
    },
    "measurement": "not_connected",
    "planning_targets": "not_connected"
  },
  "decisions": {
    "origin": "sample_seed",
    "proposals": {
      "total": 6,
      "more": false,
      "rows": [
        {
          "module": "perf_gate",
          "number": 9,
          "target": "perf-proposal-9",
          "kind": "retire",
          "state": "pending",
          "created_at": "2026-11-17T18:09:16Z",
          "proposer": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
          "approver": null,
          "rationale": "Sample data",
          "closed_by": null,
          "closing_rationale": "",
          "summary": {"profile": "sample-month"},
          "can_approve": true,
          "approve_blocked": "",
          "can_withdraw": false,
          "withdraw_blocked": "not_proposer"
        },
        {
          "module": "perf_gate",
          "number": 8,
          "target": "perf-proposal-8",
          "kind": "harness",
          "state": "applied",
          "created_at": "2026-11-17T18:09:16Z",
          "proposer": {"id": "sample-evidence-reviewer-1", "login": null, "directory": false},
          "approver": {"id": "sample-evidence-reviewer-2", "login": null, "directory": false},
          "rationale": "Sample data",
          "closed_by": {"id": "sample-evidence-reviewer-2", "login": null, "directory": false},
          "closing_rationale": "Sample data",
          "summary": {"harness": "sample-perf-harness"},
          "can_approve": false,
          "approve_blocked": "not_pending",
          "can_withdraw": false,
          "withdraw_blocked": "not_pending"
        },
        {
          "module": "perf_gate",
          "number": 7,
          "target": "perf-proposal-7",
          "kind": "budget",
          "state": "applied",
          "created_at": "2026-11-17T18:09:16Z",
          "proposer": {"id": "sample-admin", "login": "admin@example.invalid", "directory": true},
          "approver": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
          "rationale": "Sample data",
          "closed_by": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
          "closing_rationale": "Sample data",
          "summary": {"budget": "sample-budget-daily", "hardware": "sample-hw-small", "profile": "sample-daily"},
          "can_approve": false,
          "approve_blocked": "not_pending",
          "can_withdraw": false,
          "withdraw_blocked": "not_pending"
        },
        {
          "module": "release_trace",
          "number": 19,
          "target": "trace-proposal-19",
          "kind": "retire",
          "state": "pending",
          "created_at": "2026-11-17T18:09:16Z",
          "proposer": {"id": "sample-admin", "login": "admin@example.invalid", "directory": true},
          "approver": null,
          "rationale": "Sample data",
          "closed_by": null,
          "closing_rationale": "",
          "summary": {"label": "sample/sample-legacy-report", "reason": "Sample scope change", "replacement": null},
          "can_approve": false,
          "approve_blocked": "own_proposal",
          "can_withdraw": true,
          "withdraw_blocked": ""
        },
        {
          "module": "release_trace",
          "number": 18,
          "target": "trace-proposal-18",
          "kind": "reconcile",
          "state": "applied",
          "created_at": "2026-11-17T18:09:16Z",
          "proposer": {"id": "sample-admin", "login": "admin@example.invalid", "directory": true},
          "approver": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
          "rationale": "Sample data",
          "closed_by": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
          "closing_rationale": "Sample data",
          "summary": {"differences": 3, "matched": false, "path": "sample-path-xlsx", "period": "2026-10"},
          "can_approve": false,
          "approve_blocked": "not_pending",
          "can_withdraw": false,
          "withdraw_blocked": "not_pending"
        },
        {
          "module": "release_trace",
          "number": 1,
          "target": "trace-proposal-1",
          "kind": "requirement",
          "state": "applied",
          "created_at": "2026-11-17T18:09:16Z",
          "proposer": {"id": "sample-admin", "login": "admin@example.invalid", "directory": true},
          "approver": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
          "rationale": "Sample data",
          "closed_by": {"id": "sample-admin-2", "login": "admin2@example.invalid", "directory": true},
          "closing_rationale": "Sample data",
          "summary": {"label": "sample/sample-common-xlsx"},
          "can_approve": false,
          "approve_blocked": "not_pending",
          "can_withdraw": false,
          "withdraw_blocked": "not_pending"
        }
      ]
    },
    "support_decisions": "not_connected",
    "activation_review": "not_connected",
    "ledger": "in_memory"
  }
};
