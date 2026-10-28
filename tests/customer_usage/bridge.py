from runtime import isolated
from modules.focus_import.customer_usage import main


with isolated():
    raise SystemExit(main())
