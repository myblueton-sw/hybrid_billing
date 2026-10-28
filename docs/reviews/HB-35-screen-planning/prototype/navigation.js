const screens = Array.from(document.querySelectorAll('.screen'));
function showScreen() {
  const requested = location.hash.slice(1);
  const target = requested === 'usage-question' ? 'usage' : requested;
  const current = screens.find(screen => screen.id === target) || screens[0];
  screens.forEach(screen => { screen.hidden = screen !== current; });
  document.title = current.dataset.title + ' | Hybrid Billing design';
  current.setAttribute('tabindex', '-1');
  current.focus({preventScroll: true});
  window.scrollTo(0, 0);
  const dialog = document.querySelector('#inquiry-dialog');
  if (dialog && requested === 'usage-question' && !dialog.open) dialog.showModal();
}
window.addEventListener('hashchange', showScreen);
showScreen();
const search = document.querySelector('[data-customer-search]');
if (search) {
  search.addEventListener('input', () => {
    const term = search.value.toLowerCase().trim();
    const rows = Array.from(document.querySelectorAll('[data-customer-row]'));
    rows.forEach(row => { row.hidden = !row.textContent.toLowerCase().includes(term); });
    const count = rows.filter(row => !row.hidden).length;
    document.querySelector('[data-search-result]').textContent = count ? count + ' customer-periods shown' : 'No matching customers. Clear the filter to view all six.';
  });
}

const inquiryDialog = document.querySelector('#inquiry-dialog');
if (inquiryDialog) inquiryDialog.addEventListener('close', () => { location.hash = 'usage'; });
