import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export type Words = Record<string, [string, string, string]>;

const lri = (s: string) => "\u2066" + s + "\u2069";

export const unknownCode = (l: Locale, code: string) => t(l, `Unknown (${lri(code)})`, `알 수 없음(${lri(code)})`, `غير معروف (${lri(code)})`);

export function has(table: object, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(table, key);
}

export function words(l: Locale, table: Words, key: string): string {
  const w = has(table, key) ? table[key] : undefined;
  return w ? t(l, w[0], w[1], w[2]) : unknownCode(l, key);
}

export const categories = ["amount", "approval", "bulk", "duplicate", "isolation", "recovery"];
export const gates = ["e2e", "daily_inflow", "monthly_close", "interrupt_resume", "catch_up"];
export const axes = ["rows_run", "rows_day", "rows_month", "customers", "accounts", "shared_pools", "skew_permille", "fan_out", "concurrent_closes", "peak_window_minutes", "retention_months", "failure_injections", "backlog_rows"];
export const metrics = ["max_rss_bytes", "db_io_bytes", "wal_bytes", "lock_wait_ms", "p95_ms", "p99_ms", "throughput_rows_s", "completion_ms", "resume_ms"];
export const failingVerdicts = ["other_data", "incomplete", "incorrect", "revoked", "self_verified", "missing_metric", "other_hardware", "over_budget"];

export const requirementStateWords: Words = {
  designed: ["Designed, not implemented", "설계됨(미구현)", "مُصمَّم، غير منفَّذ"],
  implemented: ["Implemented, not tested", "구현됨(미시험)", "منفَّذ، لم يُختبر"],
  failing: ["Failing at this build", "이 빌드에서 실패", "فاشل في هذه البنية"],
  tested: ["Tested at this build", "이 빌드에서 시험됨", "مُختبَر في هذه البنية"],
};

export const pathStateWords: Words = {
  designed: ["Designed, not active", "설계됨(개통 안 됨)", "مُصمَّم، غير مفعّل"],
};

export const categoryWords: Words = {
  amount: ["Amounts", "금액", "المبالغ"],
  approval: ["Approval", "승인", "الاعتماد"],
  bulk: ["Bulk processing", "대량 처리", "المعالجة المجمّعة"],
  duplicate: ["Duplicates", "중복", "التكرار"],
  isolation: ["Isolation", "격리", "العزل"],
  recovery: ["Recovery", "복구", "الاسترداد"],
};

export const problemWords: Words = {
  not_tested: ["Not tested at this build", "이 빌드에서 시험 안 됨", "لم يُختبر في هذه البنية"],
  blocking: ["Blocking requirement not tested", "차단 요구사항 시험 안 됨", "متطلب حاجب لم يُختبر"],
  retired: ["Retired requirement in the candidate", "후보에 폐지된 요구사항 포함", "متطلب منتهٍ العمل به ضمن المرشّح"],
  common_missing: ["Common requirement missing from the candidate", "후보에 공통 요구사항 누락", "متطلب مشترك غير مُدرج في المرشّح"],
  common_unregistered: ["No common requirement registered", "등록된 공통 요구사항 없음", "لا يوجد متطلب مشترك مسجّل"],
  unknown: ["Unknown requirement or combination", "알 수 없는 요구사항 또는 조합", "متطلب أو مجموعة غير معروفة"],
  path_inactive: ["Combination not active", "조합 미개통", "المجموعة غير مفعّلة"],
  path_reconciliation: ["Latest reconciliation not matched", "최근 대사 불일치", "آخر مطابقة غير متطابقة"],
  path_requirement: ["Combination requirement missing from the candidate", "후보에 조합 요구사항 누락", "متطلب المجموعة غير مُدرج في المرشّح"],
  path_evidence: ["Test evidence incomplete for the combination", "조합의 시험 증거 불충분", "أدلة الاختبار غير مكتملة للمجموعة"],
};

export const outcomeWords: Words = {
  pass: ["Pass recorded", "통과 기록됨", "سُجّل نجاح"],
  fail: ["Fail recorded", "실패 기록됨", "سُجّل إخفاق"],
};

export const reasonWords: Words = {
  no_result_recorded: ["Not run: no result is recorded for this build", "미실행: 이 빌드의 결과 기록 없음", "لم يُنفَّذ: لا توجد نتيجة مسجّلة لهذه البنية"],
  no_implementation: ["Not counted: no implementation is recorded", "미반영: 구현 기록 없음", "غير محتسب: لا يوجد تنفيذ مسجّل"],
  build_predates_implementation: ["Not counted: this build was registered before the latest implementation", "미반영: 최근 구현 전에 등록된 빌드", "غير محتسب: سُجّلت البنية قبل آخر تنفيذ"],
  not_counted_by_module: ["Not counted by the release trace register", "미반영: 출시 추적 레지스트리가 인정하지 않음", "غير محتسب لدى سجل تتبع الإصدار"],
};

export const moduleWords: Words = {
  release_trace: ["Release trace", "출시 추적", "تتبع الإصدار"],
  perf_gate: ["Load", "부하", "الحمل"],
};

export const gateWords: Words = {
  e2e: ["End-to-end run", "종단 간 실행", "تشغيل من البداية إلى النهاية"],
  daily_inflow: ["Daily inflow", "일 유입", "التدفق اليومي"],
  monthly_close: ["Monthly close", "월 마감", "الإقفال الشهري"],
  interrupt_resume: ["Interruption and resume", "중단·재개", "الانقطاع والاستئناف"],
  catch_up: ["Catch-up", "지연분 처리", "تعويض التأخر"],
};

export const gateStatusWords: Words = {
  not_run: ["Not run", "미실행", "لم يُنفَّذ"],
  failing: ["Failing", "실패", "فاشل"],
  provisional: ["Provisional", "잠정", "مبدئي"],
};

export const verdictWords: Words = {
  provisional: ["Provisional: within a proposed performance budget, not approved", "잠정: 제안된 성능 예산 이내(미승인)", "مبدئي: ضمن ميزانية أداء مقترحة غير معتمدة"],
  over_budget: ["Over performance budget", "성능 예산 초과", "تجاوز ميزانية الأداء"],
  incomplete: ["Incomplete: fewer rows than the profile", "불완전: 프로파일보다 행 수가 적음", "غير مكتمل: صفوف أقل من ملف التعريف"],
  incorrect: ["Incorrect output", "출력 불일치", "مخرجات غير صحيحة"],
  other_data: ["Different dataset", "다른 데이터 세트", "مجموعة بيانات مختلفة"],
  revoked: ["Harness revoked", "실행 도구 해지됨", "أداة الاختبار مُبطَلة"],
  self_verified: ["Self-verified: not accepted", "자체 검증: 인정 안 됨", "تحقق ذاتي: غير مقبول"],
  missing_metric: ["Missing metric", "측정 항목 누락", "مقياس ناقص"],
  unbudgeted: ["No performance budget", "성능 예산 없음", "لا ميزانية أداء"],
  other_hardware: ["Different hardware", "다른 장비", "أجهزة مختلفة"],
  predates_budget: ["Run predates the performance budget", "성능 예산 이전 실행", "تشغيل سابق لميزانية الأداء"],
};

export const budgetStateWords: Words = {
  proposed: ["Proposed, not approved", "제안됨(미승인)", "مقترحة، غير معتمدة"],
};

export const storageWords: Words = {
  ssd: ["SSD", "SSD", "SSD"],
  hdd: ["HDD", "HDD", "HDD"],
  network: ["Network storage", "네트워크 스토리지", "تخزين شبكي"],
};

export const proposalStateWords: Words = {
  pending: ["Pending", "승인 대기", "بانتظار الاعتماد"],
  applied: ["Applied", "적용됨", "مطبّق"],
  withdrawn: ["Withdrawn", "철회됨", "مسحوب"],
  store_retired: ["No longer held", "보관 안 됨", "لم يعد محفوظًا"],
};

export const connectionWords: Words = {
  not_connected: ["Not connected", "연결되지 않음", "غير متصل"],
};

export const ledgerWords: Words = {
  in_memory: ["In memory (not durable)", "메모리 내(비내구)", "في الذاكرة (غير دائم)"],
};

export const originWords: Words = {
  sample_seed: ["Sample records", "샘플 기록", "سجلات عينة"],
  empty: ["No records", "기록 없음", "لا سجلات"],
};

export const kindWords: Words = {
  "release_trace:requirement": ["Requirement registration", "요구사항 등록", "تسجيل متطلب"],
  "release_trace:retire": ["Requirement retirement", "요구사항 폐지", "إنهاء العمل بمتطلب"],
  "release_trace:implemented": ["Implementation record", "구현 기록", "سجل تنفيذ"],
  "release_trace:build": ["Build registration", "빌드 등록", "تسجيل بنية"],
  "release_trace:harness": ["Test harness registration", "시험 실행 도구 등록", "تسجيل أداة اختبار"],
  "release_trace:path": ["Support combination registration", "지원 조합 등록", "تسجيل مجموعة دعم"],
  "release_trace:reconcile": ["Reconciliation record", "대사 기록", "سجل مطابقة"],
  "release_trace:proof": ["Customer proof (not connected)", "고객 실증(연결되지 않음)", "إثبات لدى العميل (غير متصل)"],
  "release_trace:activate": ["Activation (not connected)", "개통(연결되지 않음)", "تفعيل (غير متصل)"],
  "perf_gate:profile": ["Load profile registration", "부하 프로파일 등록", "تسجيل ملف تعريف حمل"],
  "perf_gate:hardware": ["Hardware registration", "장비 등록", "تسجيل أجهزة"],
  "perf_gate:budget": ["Performance budget proposal", "성능 예산 제안", "مقترح ميزانية أداء"],
  "perf_gate:approval": ["Performance budget approval (not connected)", "성능 예산 승인(연결되지 않음)", "اعتماد ميزانية الأداء (غير متصل)"],
  "perf_gate:harness": ["Load harness registration", "부하 실행 도구 등록", "تسجيل أداة حمل"],
  "perf_gate:retire": ["Load profile retirement", "부하 프로파일 폐지", "إنهاء العمل بملف تعريف حمل"],
};

export const summaryWords: Words = {
  "release_trace:requirement": ["Register requirement {label}", "요구사항 {label} 등록", "تسجيل المتطلب {label}"],
  "release_trace:retire:replaced": ["Retire requirement {label}, replaced by {replacement}: {reason}", "요구사항 {label} 폐지, 대체: {replacement}: {reason}", "إنهاء العمل بالمتطلب {label}، ويحل محله {replacement}: {reason}"],
  "release_trace:retire:alone": ["Retire requirement {label} with no replacement: {reason}", "요구사항 {label} 폐지, 대체 없음: {reason}", "إنهاء العمل بالمتطلب {label} دون بديل: {reason}"],
  "release_trace:implemented": ["Record the implementation of {label} at commit {commit}", "{label} 구현 기록, 커밋 {commit}", "تسجيل تنفيذ {label} عند الالتزام {commit}"],
  "release_trace:build": ["Register build {build} at commit {commit}", "빌드 {build} 등록, 커밋 {commit}", "تسجيل البنية {build} عند الالتزام {commit}"],
  "release_trace:harness": ["Register test harness {harness}", "시험 실행 도구 {harness} 등록", "تسجيل أداة الاختبار {harness}"],
  "release_trace:path": ["Register support combination {path}", "지원 조합 {path} 등록", "تسجيل مجموعة الدعم {path}"],
  "release_trace:reconcile:matched": ["Record reconciliation of {path} for {period}: matched, {differences}", "{path}의 {period} 대사 기록: 일치, {differences}", "تسجيل مطابقة {path} للفترة {period}: متطابقة، {differences}"],
  "release_trace:reconcile:unmatched": ["Record reconciliation of {path} for {period}: not matched, {differences}", "{path}의 {period} 대사 기록: 불일치, {differences}", "تسجيل مطابقة {path} للفترة {period}: غير متطابقة، {differences}"],
  "release_trace:proof": ["Customer proof for {label} at {build} for {period} (not connected)", "{label}의 {build} 고객 실증, 기간 {period}(연결되지 않음)", "إثبات لدى العميل لـ {label} عند {build} للفترة {period} (غير متصل)"],
  "release_trace:activate": ["Activation of {path} at {build} (not connected)", "{path}의 {build} 개통(연결되지 않음)", "تفعيل {path} عند {build} (غير متصل)"],
  "perf_gate:profile": ["Register load profile {profile} for {gate}", "{gate}용 부하 프로파일 {profile} 등록", "تسجيل ملف تعريف الحمل {profile} لـ {gate}"],
  "perf_gate:hardware": ["Register hardware {hardware}", "장비 {hardware} 등록", "تسجيل الأجهزة {hardware}"],
  "perf_gate:budget": ["Propose performance budget {budget} for {profile} on {hardware}", "{hardware}에서 {profile}의 성능 예산 {budget} 제안", "اقتراح ميزانية الأداء {budget} لـ {profile} على {hardware}"],
  "perf_gate:approval": ["Performance budget approval for {budget} (not connected)", "성능 예산 {budget} 승인(연결되지 않음)", "اعتماد ميزانية الأداء {budget} (غير متصل)"],
  "perf_gate:harness": ["Register load harness {harness}", "부하 실행 도구 {harness} 등록", "تسجيل أداة الحمل {harness}"],
  "perf_gate:retire": ["Retire load profile {profile}", "부하 프로파일 {profile} 폐지", "إنهاء العمل بملف تعريف الحمل {profile}"],
};

export const blockerWords: Words = {
  already_proposed: ["A retirement proposal is already waiting for approval.", "폐지 제안이 이미 승인을 기다리고 있습니다.", "يوجد مقترح لإنهاء العمل به بانتظار الاعتماد بالفعل."],
  no_eligible_approver: ["No other administrator could approve a retirement now, so it cannot be proposed.", "지금은 폐지를 승인할 수 있는 다른 관리자가 없어 제안할 수 없습니다.", "لا يوجد مسؤول آخر يمكنه اعتماد إنهاء العمل به الآن، لذا لا يمكن اقتراحه."],
  no_replacement: ["This requirement is common, so it needs a replacement, and no requirement can replace it now.", "공통 요구사항이라 대체가 필요하지만, 지금 대체할 수 있는 요구사항이 없습니다.", "هذا المتطلب مشترك، لذا يحتاج إلى بديل، ولا يوجد متطلب يمكن أن يحل محله الآن."],
  common_needs_replacement: ["The requirement is common and the proposal names no replacement.", "공통 요구사항인데 제안에 대체 요구사항이 없습니다.", "المتطلب مشترك والمقترح لا يحدد بديلًا."],
  weaker_replacement: ["The replacement is no longer at least as strong, or is retired or already a replacement. Withdraw and propose again.", "대체 요구사항이 더 이상 같은 수준 이상이 아니거나, 폐지되었거나, 이미 다른 대체로 지정되었습니다. 철회한 뒤 다시 제안하세요.", "لم يعد البديل بالقوة نفسها على الأقل، أو انتهى العمل به، أو هو بديل بالفعل. اسحب المقترح واقترح من جديد."],
  retired: ["The requirement or profile is already retired.", "요구사항 또는 프로파일이 이미 폐지되었습니다.", "انتهى العمل بالمتطلب أو ملف التعريف بالفعل."],
  own_proposal: ["You proposed this. Another administrator must approve it.", "직접 제안한 항목입니다. 다른 관리자가 승인해야 합니다.", "أنت من اقترح هذا. يجب أن يعتمده مسؤول آخر."],
  not_pending: ["This is no longer waiting for a decision. Reload.", "더 이상 결정을 기다리는 항목이 아닙니다. 새로 고치세요.", "لم يعد هذا بانتظار قرار. أعد التحميل."],
  store_retired: ["The register no longer holds this proposal. Nothing can be done with it here.", "레지스트리에 이 제안이 더 이상 보관되어 있지 않습니다. 여기서 할 수 있는 작업이 없습니다.", "لم يعد السجل يحتفظ بهذا المقترح. لا يمكن فعل أي شيء به هنا."],
  not_console_kind: ["Proposals of this kind cannot be approved or withdrawn from the console in this build.", "이 빌드에서는 이 종류의 제안을 콘솔에서 승인하거나 철회할 수 없습니다.", "لا يمكن اعتماد مقترحات من هذا النوع أو سحبها من وحدة التحكم في هذه البنية."],
  target_too_long: ["This profile's identifier is too long to name in a command, so it cannot be retired here.", "이 프로파일의 식별자가 명령에 넣기에 너무 길어 여기서 폐지할 수 없습니다.", "معرّف ملف التعريف هذا أطول من أن يُذكر في أمر، لذا لا يمكن إنهاء العمل به هنا."],
  not_found: ["The register no longer knows this requirement or its replacement.", "레지스트리가 이 요구사항 또는 대체 요구사항을 더 이상 알지 못합니다.", "لم يعد السجل يعرف هذا المتطلب أو بديله."],
  not_proposer: ["Only the proposer can withdraw it.", "제안자만 철회할 수 있습니다.", "يمكن للمقترِح وحده سحبه."],
};

export const stateSentenceWords: Words = {
  not_proposer: ["Only the proposer can withdraw it.", "제안자만 철회할 수 있습니다.", "يمكن للمقترِح وحده سحبه."],
  retired: ["Retired; there is nothing to retire.", "폐지됨. 폐지할 대상이 없습니다.", "انتهى العمل به؛ لا يوجد ما يُنهى."],
};

export const keptWords: Words = {
  "release_evidence.approve:refused": ["You cannot approve your own proposal. Nothing changed.", "자신의 제안은 승인할 수 없습니다. 변경된 내용이 없습니다.", "لا يمكنك اعتماد مقترحك. لم يتغيّر شيء."],
  "release_evidence.approve:stage": [
    "The register refused this approval. Nothing changed. The proposal may no longer be pending, the requirement or profile may already be retired, or the replacement may no longer be at least as strong.",
    "레지스트리가 이 승인을 거부했습니다. 변경된 내용이 없습니다. 제안이 더 이상 대기 중이 아니거나, 요구사항 또는 프로파일이 이미 폐지되었거나, 대체 요구사항이 더 이상 같은 수준 이상이 아닐 수 있습니다.",
    "رفض السجل هذا الاعتماد. لم يتغيّر شيء. ربما لم يعد المقترح بانتظار الاعتماد، أو انتهى العمل بالمتطلب أو ملف التعريف بالفعل، أو لم يعد البديل بالقوة نفسها على الأقل.",
  ],
  "release_evidence.withdraw:refused": ["Only the proposer can withdraw it. Nothing changed.", "제안자만 철회할 수 있습니다. 변경된 내용이 없습니다.", "يمكن للمقترِح وحده سحبه. لم يتغيّر شيء."],
  "release_evidence.withdraw:stage": ["The register refused this withdrawal. Nothing changed.", "레지스트리가 이 철회를 거부했습니다. 변경된 내용이 없습니다.", "رفض السجل هذا السحب. لم يتغيّر شيء."],
  "release_evidence.retire_requirement:stage": [
    "The retirement was not proposed. Nothing changed. It may already be retired or proposed, it may need a replacement, or the replacement may be too weak.",
    "폐지가 제안되지 않았습니다. 변경된 내용이 없습니다. 이미 폐지되었거나 제안되었을 수 있고, 대체가 필요하거나 대체 요구사항이 충분하지 않을 수 있습니다.",
    "لم يُقترح إنهاء العمل بالمتطلب. لم يتغيّر شيء. ربما انتهى العمل به أو اقتُرح بالفعل، أو يحتاج إلى بديل، أو البديل أضعف من اللازم.",
  ],
  "release_evidence.retire_requirement:refused": [
    "The retirement was not proposed. Nothing changed. You have too many pending proposals in the register.",
    "폐지가 제안되지 않았습니다. 변경된 내용이 없습니다. 레지스트리에 대기 중인 제안이 너무 많습니다.",
    "لم يُقترح إنهاء العمل بالمتطلب. لم يتغيّر شيء. لديك مقترحات كثيرة جدًا بانتظار الاعتماد في السجل.",
  ],
  "release_evidence.retire_profile:stage": [
    "The profile retirement was not proposed. Nothing changed. It may already be retired or proposed.",
    "프로파일 폐지가 제안되지 않았습니다. 변경된 내용이 없습니다. 이미 폐지되었거나 제안되었을 수 있습니다.",
    "لم يُقترح إنهاء العمل بملف التعريف. لم يتغيّر شيء. ربما انتهى العمل به أو اقتُرح بالفعل.",
  ],
  "release_evidence.retire_profile:refused": [
    "The profile retirement was not proposed. Nothing changed. You have too many pending proposals in the register.",
    "프로파일 폐지가 제안되지 않았습니다. 변경된 내용이 없습니다. 레지스트리에 대기 중인 제안이 너무 많습니다.",
    "لم يُقترح إنهاء العمل بملف التعريف. لم يتغيّر شيء. لديك مقترحات كثيرة جدًا بانتظار الاعتماد في السجل.",
  ],
  earlier: [
    "The register refused this request. Your earlier request may have been applied: check the list.",
    "레지스트리가 이 요청을 거부했습니다. 앞서 보낸 요청이 적용되었을 수 있습니다. 목록을 확인하세요.",
    "رفض السجل هذا الطلب. ربما طُبّق طلبك السابق: تحقّق من القائمة.",
  ],
};

export const axisWords: Words = {
  rows_run: ["Rows a run", "실행당 행 수", "صفوف لكل تشغيل"],
  rows_day: ["Rows a day", "일 행 수", "صفوف يوميًا"],
  rows_month: ["Rows a month", "월 행 수", "صفوف شهريًا"],
  customers: ["Customers", "고객 수", "العملاء"],
  accounts: ["Accounts", "계정 수", "الحسابات"],
  shared_pools: ["Shared pools", "공유 풀 수", "المجمّعات المشتركة"],
  skew_permille: ["Skew (per mille)", "편중(천분율)", "الانحراف (في الألف)"],
  fan_out: ["Fan-out", "분기 수", "التفرّع"],
  concurrent_closes: ["Concurrent closes", "동시 마감 수", "الإقفالات المتزامنة"],
  peak_window_minutes: ["Peak window (minutes)", "피크 구간(분)", "نافذة الذروة (دقائق)"],
  retention_months: ["Retention (months)", "보관 기간(개월)", "الاحتفاظ (أشهر)"],
  failure_injections: ["Failure injections", "장애 주입 수", "حقن الأعطال"],
  backlog_rows: ["Backlog rows", "지연분 행 수", "صفوف المتأخرات"],
};

export const metricWords: Words = {
  max_rss_bytes: ["Peak memory (bytes)", "최대 메모리(바이트)", "ذروة الذاكرة (بايت)"],
  db_io_bytes: ["Database I/O (bytes)", "데이터베이스 I/O(바이트)", "إدخال/إخراج قاعدة البيانات (بايت)"],
  wal_bytes: ["Write-ahead log (bytes)", "WAL(미리 쓰기 로그, 바이트)", "سجل الكتابة المسبقة (بايت)"],
  lock_wait_ms: ["Lock wait (ms)", "잠금 대기(ms)", "انتظار القفل (مللي ثانية)"],
  p95_ms: ["95th percentile latency (ms)", "95번째 백분위 지연(ms)", "زمن الاستجابة عند المئين 95 (مللي ثانية)"],
  p99_ms: ["99th percentile latency (ms)", "99번째 백분위 지연(ms)", "زمن الاستجابة عند المئين 99 (مللي ثانية)"],
  throughput_rows_s: ["Throughput (rows/s)", "처리량(행/초)", "معدل المعالجة (صفوف/ثانية)"],
  completion_ms: ["Completion (ms)", "완료 시간(ms)", "زمن الإكمال (مللي ثانية)"],
  resume_ms: ["Resume (ms)", "재개 시간(ms)", "زمن الاستئناف (مللي ثانية)"],
};

export type Plural = { en: [string, string]; ko: string; ar: [string, string, string, string, string, string] };

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };
const arOrder = ["zero", "one", "two", "few", "many", "other"];

export function category(l: Locale, n: number): string {
  return new Intl.PluralRules(tags[l]).select(n);
}

export function fill(base: string, slots: Record<string, string>): string {
  return Object.entries(slots).reduce((s, [key, v]) => s.split("{" + key + "}").join(v), base);
}

export function counted(l: Locale, n: number, forms: Plural, slots: Record<string, string> = {}): string {
  const c = category(l, n);
  const base = l === "en" ? (c === "one" ? forms.en[0] : forms.en[1]) : l === "ko" ? forms.ko : forms.ar[Math.max(0, arOrder.indexOf(c))];
  return fill(base, { n: String(n), ...slots });
}

export const plurals = {
  problems: {
    en: ["1 problem", "{n} problems"],
    ko: "문제 {n}건",
    ar: ["لا مشكلات", "مشكلة واحدة", "مشكلتان", "{n} مشكلات", "{n} مشكلة", "{n} مشكلة"],
  },
  gateNotReady: {
    en: ["Gate check for {version} at {build}: not ready, 1 problem.", "Gate check for {version} at {build}: not ready, {n} problems."],
    ko: "{version}의 빌드 {build} 게이트 점검: 준비되지 않음, 문제 {n}건.",
    ar: [
      "فحص البوابة لـ {version} عند {build}: غير جاهز، لا مشكلات.",
      "فحص البوابة لـ {version} عند {build}: غير جاهز، مشكلة واحدة.",
      "فحص البوابة لـ {version} عند {build}: غير جاهز، مشكلتان.",
      "فحص البوابة لـ {version} عند {build}: غير جاهز، {n} مشكلات.",
      "فحص البوابة لـ {version} عند {build}: غير جاهز، {n} مشكلة.",
      "فحص البوابة لـ {version} عند {build}: غير جاهز، {n} مشكلة.",
    ],
  },
  showing50: {
    en: ["Showing 50 of 1.", "Showing 50 of {n}."],
    ko: "{n}건 중 50건을 표시합니다.",
    ar: ["عرض 50 من أصل {n}.", "عرض 50 من أصل عنصر واحد.", "عرض 50 من أصل عنصرين.", "عرض 50 من أصل {n} عناصر.", "عرض 50 من أصل {n} عنصرًا.", "عرض 50 من أصل {n} عنصر."],
  },
  showingNewest50: {
    en: ["Showing the newest 50 of 1.", "Showing the newest 50 of {n}."],
    ko: "{n}건 중 최근 50건을 표시합니다.",
    ar: ["عرض أحدث 50 من أصل {n}.", "عرض أحدث 50 من أصل عنصر واحد.", "عرض أحدث 50 من أصل عنصرين.", "عرض أحدث 50 من أصل {n} عناصر.", "عرض أحدث 50 من أصل {n} عنصرًا.", "عرض أحدث 50 من أصل {n} عنصر."],
  },
  showingNewest20: {
    en: ["Showing the newest 20 of 1.", "Showing the newest 20 of {n}."],
    ko: "{n}건 중 최근 20건을 표시합니다.",
    ar: ["عرض أحدث 20 من أصل {n}.", "عرض أحدث 20 من أصل عنصر واحد.", "عرض أحدث 20 من أصل عنصرين.", "عرض أحدث 20 من أصل {n} عناصر.", "عرض أحدث 20 من أصل {n} عنصرًا.", "عرض أحدث 20 من أصل {n} عنصر."],
  },
  showingNewest12: {
    en: ["Showing the newest 12 of 1.", "Showing the newest 12 of {n}."],
    ko: "{n}건 중 최근 12건을 표시합니다.",
    ar: ["عرض أحدث 12 من أصل {n}.", "عرض أحدث 12 من أصل عنصر واحد.", "عرض أحدث 12 من أصل عنصرين.", "عرض أحدث 12 من أصل {n} عناصر.", "عرض أحدث 12 من أصل {n} عنصرًا.", "عرض أحدث 12 من أصل {n} عنصر."],
  },
  showing100: {
    en: ["Showing 100 of 1.", "Showing 100 of {n}."],
    ko: "{n}건 중 100건을 표시합니다.",
    ar: ["عرض 100 من أصل {n}.", "عرض 100 من أصل عنصر واحد.", "عرض 100 من أصل عنصرين.", "عرض 100 من أصل {n} عناصر.", "عرض 100 من أصل {n} عنصرًا.", "عرض 100 من أصل {n} عنصر."],
  },
  failing: {
    en: ["1 failing", "{n} failing"],
    ko: "실패 {n}건",
    ar: ["لا إخفاقات", "إخفاق واحد", "إخفاقان", "{n} إخفاقات", "{n} إخفاقًا", "{n} إخفاق"],
  },
  differences: {
    en: ["1 difference", "{n} differences"],
    ko: "차이 {n}건",
    ar: ["لا فروق", "فرق واحد", "فرقان", "{n} فروق", "{n} فرقًا", "{n} فرق"],
  },
  tests: {
    en: ["1 test", "{n} tests"],
    ko: "시험 {n}건",
    ar: ["لا اختبارات", "اختبار واحد", "اختباران", "{n} اختبارات", "{n} اختبارًا", "{n} اختبار"],
  },
  runs: {
    en: ["1 run", "{n} runs"],
    ko: "실행 {n}건",
    ar: ["لا عمليات تشغيل", "تشغيل واحد", "تشغيلان", "{n} عمليات تشغيل", "{n} عملية تشغيل", "{n} عملية تشغيل"],
  },
} satisfies Record<string, Plural>;

export const allWordTables: Record<string, Words> = {
  requirementStateWords, pathStateWords, categoryWords, problemWords, outcomeWords, reasonWords, moduleWords, gateWords, gateStatusWords, verdictWords, budgetStateWords, storageWords,
  proposalStateWords, connectionWords, ledgerWords, originWords, kindWords, summaryWords, blockerWords, stateSentenceWords, keptWords, axisWords, metricWords,
};
