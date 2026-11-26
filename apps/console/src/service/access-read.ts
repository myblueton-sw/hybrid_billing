import type { AccessSetting, Preview, Ceiling } from "./admin";
import { array, boolean, integer, shape, strings, text } from "./strict-read";

export const catalogOf = array(shape<Ceiling>({ action: text, kind: text, fields: strings, until: text }));
export const settingsOf = array(shape<AccessSetting>({ name: text, value: integer, unit: text, default: integer, min: integer, max: integer, source: text }));
export const previewsOf = array(shape<Preview>({ action: text, field: text, allowed: boolean, reason: text, delegations: strings }));
