import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export type Words = Record<string, [string, string, string]>;

export const unknownCode = (l: Locale, code: string) => t(l, `Unknown (${code})`, `알 수 없음(${code})`, `غير معروف (${code})`);

export function words(l: Locale, table: Words, key: string, fallback: (k: string) => string = (k) => unknownCode(l, k)): string {
  const w = Object.prototype.hasOwnProperty.call(table, key) ? table[key] : undefined;
  return w ? t(l, w[0], w[1], w[2]) : fallback(key);
}

export const proposalKindWords: Words = {
  key: ["Release signing key", "릴리스 서명 키", "مفتاح توقيع الإصدار"],
  release: ["Release", "릴리스", "إصدار"],
  site: ["Site", "설치 사이트", "الموقع"],
  probe: ["Site probe", "사이트 프로브", "مسبار الموقع"],
  config: ["Site configuration", "사이트 설정", "إعدادات الموقع"],
  destination: ["Allowed destination", "허용 목적지", "وجهة مسموح بها"],
  plan: ["Install plan acceptance", "설치 계획 수락", "قبول خطة التثبيت"],
  upgrade: ["Upgrade plan acceptance", "업그레이드 계획 수락", "قبول خطة الترقية"],
  rollback: ["Rollback plan acceptance", "롤백 계획 수락", "قبول خطة التراجع"],
};

export const summaryWords: Words = {
  key: ["Register release signing key", "릴리스 서명 키 등록", "تسجيل مفتاح توقيع الإصدار"],
  release: ["Register release", "릴리스 등록", "تسجيل الإصدار"],
  site: ["Register site", "설치 사이트 등록", "تسجيل الموقع"],
  probe: ["Change the probe of site", "사이트 프로브 변경", "تغيير مسبار الموقع"],
  config: ["Change the configuration of site", "사이트 설정 변경", "تغيير إعدادات الموقع"],
  destination: ["Allow destination", "목적지 허용", "السماح بالوجهة"],
  plan: ["Accept install plan", "설치 계획 수락", "قبول خطة التثبيت"],
  upgrade: ["Accept upgrade plan", "업그레이드 계획 수락", "قبول خطة الترقية"],
  rollback: ["Accept rollback plan", "롤백 계획 수락", "قبول خطة التراجع"],
};

export const proposalStateWords: Words = {
  pending: ["Pending", "승인 대기", "بانتظار الاعتماد"],
  applied: ["Applied", "적용됨", "مطبّق"],
  withdrawn: ["Withdrawn", "철회됨", "مسحوب"],
};

export const observationWords: Words = {
  fresh: ["Fresh", "최신", "حديثة"],
  stale: ["Stale", "오래됨", "قديمة"],
  missing: ["Missing", "없음", "غير موجودة"],
  unknown: ["Unknown", "알 수 없음", "غير معروفة"],
};

export const installedWords: Words = {
  installed: ["Installed", "설치됨", "مثبّت"],
  upgrade_failed: ["Upgrade failed", "업그레이드 실패", "فشلت الترقية"],
  not_installed: ["Not installed", "설치되지 않음", "غير مثبّت"],
};

export const actionWords: Words = {
  create: ["Create", "생성", "إنشاء"],
  bind: ["Bind", "포트 연결", "ربط"],
  replace: ["Replace", "교체", "استبدال"],
  remove: ["Remove", "제거", "إزالة"],
  reuse: ["Reuse", "재사용", "إعادة استخدام"],
  retain: ["Keep", "유지", "إبقاء"],
  detach: ["Detach", "분리", "فصل"],
  reattached: ["Reattach", "다시 연결", "إعادة إرفاق"],
  adopt_created: ["Adopt as created", "생성된 것으로 등록", "اعتماده كمُنشأ"],
  restart: ["Restart", "다시 시작", "إعادة تشغيل"],
  leftover: ["Left behind", "남겨 둠", "يبقى بعد العملية"],
};

export const conflictWords: Words = {
  port_moved: ["The port moved", "포트가 바뀜", "تغيّر المنفذ"],
  volume_remapped: ["The volume was mapped elsewhere", "볼륨 위치가 바뀜", "أُعيد تعيين وحدة التخزين"],
  data_volume_removed: ["A data volume would be removed", "데이터 볼륨이 제거됨", "ستُزال وحدة تخزين بيانات"],
  owned_missing: ["An installed item is missing", "설치된 항목이 없음", "عنصر مثبّت مفقود"],
  service_exists: ["A service with this name already exists", "같은 이름의 서비스가 이미 있음", "توجد خدمة بهذا الاسم"],
  port_in_use: ["The port is already in use", "포트가 이미 사용 중", "المنفذ مستخدم بالفعل"],
  port_duplicate: ["The port is used twice in this plan", "이 계획에서 포트가 중복됨", "المنفذ مكرر في هذه الخطة"],
  existing_volume: ["A volume already exists at this location", "이 위치에 볼륨이 이미 있음", "توجد وحدة تخزين في هذا الموقع"],
  volume_duplicate: ["The volume location is used twice in this plan", "이 계획에서 볼륨 위치가 중복됨", "موقع وحدة التخزين مكرر في هذه الخطة"],
  config_mismatch: ["The site configuration is for another version", "사이트 설정이 다른 버전용임", "إعدادات الموقع لإصدار آخر"],
};

export const effectWords: Words = {
  new: ["New data volume", "새 데이터 볼륨", "وحدة تخزين بيانات جديدة"],
  untouched: ["Existing content left untouched", "기존 내용은 건드리지 않음", "يبقى المحتوى الحالي دون تغيير"],
  retained: ["Data kept", "데이터 유지", "تُحفظ البيانات"],
  stale_content: ["Content may be out of date", "내용이 오래되었을 수 있음", "قد يكون المحتوى قديمًا"],
};

export const permissionWords: Words = {
  run_service: ["Run services", "서비스 실행", "تشغيل الخدمات"],
  write_config: ["Write configuration", "설정 쓰기", "كتابة الإعدادات"],
  listen_port: ["Listen on ports", "포트 수신", "الاستماع على المنافذ"],
  create_volume: ["Create volumes", "볼륨 생성", "إنشاء وحدات التخزين"],
};

export const artifactWords: Words = {
  container_image: ["Container image", "컨테이너 이미지", "صورة حاوية"],
  archive: ["Archive", "압축 파일", "أرشيف"],
  package: ["Package", "패키지", "حزمة"],
};

export const reasonWords: Words = {
  not_observed: ["the site has not been observed", "사이트를 아직 관측하지 않았습니다", "لم تتم مراقبة الموقع"],
  observation_stale: ["the site observation is stale", "사이트 관측이 오래되었습니다", "مراقبة الموقع قديمة"],
  already_installed: ["this version is already installed", "이 버전은 이미 설치되어 있습니다", "هذا الإصدار مثبّت بالفعل"],
  unsupported_platform: ["the release does not support the site platform", "릴리스가 사이트 플랫폼을 지원하지 않습니다", "الإصدار لا يدعم منصة الموقع"],
  platform_mismatch: ["the observed platform does not match the site", "관측한 플랫폼이 사이트와 다릅니다", "المنصة المرصودة لا تطابق الموقع"],
  incomplete: ["the site observation is incomplete", "사이트 관측이 완전하지 않습니다", "مراقبة الموقع غير مكتملة"],
  revoked: ["the release or its key is revoked", "릴리스나 서명 키가 폐기되었습니다", "الإصدار أو مفتاحه ملغى"],
  release_revoked: ["the release is revoked", "릴리스가 폐기되었습니다", "الإصدار ملغى"],
  not_installed: ["nothing is installed", "설치된 것이 없습니다", "لا شيء مثبّت"],
  not_upgradable: ["the installed version cannot be upgraded to this release", "설치된 버전에서 이 릴리스로 업그레이드할 수 없습니다", "لا يمكن ترقية الإصدار المثبّت إلى هذا الإصدار"],
  direction: ["this release is not newer than the installed version", "이 릴리스는 설치된 버전보다 새롭지 않습니다", "هذا الإصدار ليس أحدث من المثبّت"],
  data_form: ["the data format does not allow this change", "데이터 형식상 이 변경을 할 수 없습니다", "تنسيق البيانات لا يسمح بهذا التغيير"],
  data_touched: ["data changed after the upgrade", "업그레이드 후 데이터가 바뀌었습니다", "تغيّرت البيانات بعد الترقية"],
  not_needed: ["nothing needs to change", "바꿀 것이 없습니다", "لا يلزم أي تغيير"],
  rolled_back: ["it was already rolled back", "이미 롤백했습니다", "تم التراجع عنه بالفعل"],
  run_open: ["an installer run is open", "진행 중인 설치 실행이 있습니다", "يوجد تشغيل مثبّت مفتوح"],
  interrupted: ["an installer run was interrupted", "설치 실행이 중단되었습니다", "انقطع تشغيل المثبّت"],
  not_declared: ["the release does not declare it", "릴리스에 선언되지 않았습니다", "الإصدار لا يعلن عنه"],
  conflicts: ["the plan has conflicts", "계획에 충돌이 있습니다", "الخطة فيها تعارضات"],
  plan_changed: ["the install plan changed", "설치 계획이 바뀌었습니다", "تغيّرت خطة التثبيت"],
  dependency_unavailable: ["the installer is unavailable", "설치 관리자를 사용할 수 없습니다", "المثبّت غير متاح"],
  not_allowed_state: ["the installed state does not allow it", "현재 설치 상태에서는 할 수 없습니다", "حالة التثبيت لا تسمح بذلك"],
  key_inactive: ["the signing key is not active", "서명 키가 활성 상태가 아닙니다", "مفتاح التوقيع غير نشط"],
  self_approval: ["you proposed it", "본인이 제안했습니다", "أنت من اقترحه"],
};

export const checkWords: Words = {
  dns: ["Name resolution (DNS)", "이름 확인(DNS)", "تحليل الاسم (DNS)"],
  tcp: ["Network connection (TCP)", "네트워크 연결(TCP)", "اتصال الشبكة (TCP)"],
  proxy: ["Proxy", "프록시", "الوكيل"],
  tls: ["Certificate (TLS)", "인증서(TLS)", "الشهادة (TLS)"],
};

export const failureWords: Words = {
  ok: ["Passed – no action", "통과 – 조치 없음", "نجح – لا إجراء"],
  unknown: ["Unclassified failure", "분류되지 않은 실패", "فشل غير مصنّف"],
  unknown_authority: ["Untrusted certificate issuer", "신뢰하지 않는 인증서 발급자", "جهة إصدار شهادة غير موثوقة"],
  hostname_mismatch: ["Certificate name does not match", "인증서 이름 불일치", "اسم الشهادة غير مطابق"],
  expired: ["Certificate expired", "인증서 만료", "الشهادة منتهية"],
  not_yet_valid: ["Certificate not yet valid", "인증서 유효 기간 전", "الشهادة غير صالحة بعد"],
  weak_signature: ["Weak certificate signature", "취약한 인증서 서명", "توقيع شهادة ضعيف"],
  proxy_auth: ["Proxy sign-in required", "프록시 인증 필요", "الوكيل يتطلب مصادقة"],
  proxy_refused: ["Proxy refused the connection", "프록시가 연결을 거부함", "رفض الوكيل الاتصال"],
  dns_failure: ["Name not resolved", "이름을 확인하지 못함", "تعذّر تحليل الاسم"],
  refused: ["Connection refused", "연결 거부됨", "رُفض الاتصال"],
  timeout: ["Connection timed out", "연결 시간 초과", "انتهت مهلة الاتصال"],
};

export const causeWords: Words = {
  none: ["No problem found.", "문제가 없습니다.", "لم يُعثر على مشكلة."],
  unclassified: ["The probe could not classify this failure.", "프로브가 이 실패를 분류하지 못했습니다.", "تعذّر على المسبار تصنيف هذا الفشل."],
  untrusted_issuer: ["The certificate was issued by an authority the product does not trust.", "제품이 신뢰하지 않는 기관이 인증서를 발급했습니다.", "أصدرت الشهادة جهة لا يثق بها المنتج."],
  certificate_name: ["The certificate does not name this host.", "인증서에 이 호스트 이름이 없습니다.", "الشهادة لا تتضمن اسم هذا المضيف."],
  validity_window: ["The certificate is outside its validity period, or a clock is wrong.", "인증서 유효 기간을 벗어났거나 시계가 맞지 않습니다.", "الشهادة خارج فترة صلاحيتها أو أن الساعة غير صحيحة."],
  weak_algorithm: ["The certificate uses a weak signature algorithm.", "인증서가 취약한 서명 알고리즘을 씁니다.", "تستخدم الشهادة خوارزمية توقيع ضعيفة."],
  proxy_credentials: ["The proxy needs credentials the product does not have.", "프록시에 제품이 갖고 있지 않은 인증 정보가 필요합니다.", "يحتاج الوكيل إلى بيانات اعتماد لا يملكها المنتج."],
  proxy_policy: ["The proxy policy does not allow this destination.", "프록시 정책이 이 목적지를 허용하지 않습니다.", "سياسة الوكيل لا تسمح بهذه الوجهة."],
  name_resolution: ["The host name could not be resolved.", "호스트 이름을 확인하지 못했습니다.", "تعذّر تحليل اسم المضيف."],
  network_path: ["The network path to the destination is blocked or the service is not listening.", "목적지로 가는 네트워크 경로가 막혔거나 서비스가 수신하지 않습니다.", "مسار الشبكة إلى الوجهة محجوب أو أن الخدمة لا تستمع."],
};

export const stepWords: Words = {
  collect_details: ["Collect the probe details and contact support.", "프로브 세부 정보를 모아 지원팀에 문의하세요.", "اجمع تفاصيل المسبار وتواصل مع الدعم."],
  add_internal_ca_to_product_trust: [
    "Add your internal certificate authority to the product's trust store. Keep certificate verification on.",
    "내부 인증 기관을 제품의 신뢰 저장소에 추가하세요. 인증서 검증은 계속 켜 두세요.",
    "أضف جهة الشهادات الداخلية إلى مخزن الثقة في المنتج. أبقِ التحقق من الشهادات مفعّلًا.",
  ],
  verify_chain: ["Check that the server sends its full certificate chain.", "서버가 전체 인증서 체인을 보내는지 확인하세요.", "تحقّق من أن الخادم يرسل سلسلة الشهادات كاملة."],
  use_certificate_hostname: ["Use the host name written in the certificate.", "인증서에 적힌 호스트 이름을 쓰세요.", "استخدم اسم المضيف المكتوب في الشهادة."],
  reissue_certificate: ["Ask the destination owner to reissue the certificate.", "목적지 관리자에게 인증서 재발급을 요청하세요.", "اطلب من مالك الوجهة إعادة إصدار الشهادة."],
  check_clock: ["Check the clock on the site and on the destination.", "사이트와 목적지의 시계를 확인하세요.", "تحقّق من الساعة في الموقع وفي الوجهة."],
  renew_certificate: ["Ask the destination owner to renew the certificate.", "목적지 관리자에게 인증서 갱신을 요청하세요.", "اطلب من مالك الوجهة تجديد الشهادة."],
  configure_proxy_credentials: ["Configure the proxy credentials for the product.", "제품에 프록시 인증 정보를 설정하세요.", "اضبط بيانات اعتماد الوكيل للمنتج."],
  configure_proxy_allowlist: ["Ask the proxy owner to allow this destination.", "프록시 관리자에게 이 목적지 허용을 요청하세요.", "اطلب من مالك الوكيل السماح بهذه الوجهة."],
  check_dns: ["Check the DNS records and the site's DNS servers.", "DNS 레코드와 사이트의 DNS 서버를 확인하세요.", "تحقّق من سجلات DNS وخوادم DNS في الموقع."],
  check_firewall: ["Check the firewall rules between the site and the destination.", "사이트와 목적지 사이의 방화벽 규칙을 확인하세요.", "تحقّق من قواعد الجدار الناري بين الموقع والوجهة."],
  check_service: ["Check that the destination service is running.", "목적지 서비스가 실행 중인지 확인하세요.", "تحقّق من أن خدمة الوجهة تعمل."],
};
