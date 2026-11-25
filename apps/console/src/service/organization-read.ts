import type { OrgPage, OrgParty, OrgProfile, OrgMember, OrgTreeNode, OrgTreePage } from "./admin";
import { array, boolean, choice, integer, shape, text } from "./strict-read";

const party = shape<OrgParty>({ id: text, name: text, kind: choice("legal_entity", "internal_unit"), parent: text, state: text });
const profile = shape<OrgProfile>({ id: text, name: text, issuer: text, buyer: text, payer: text, recipient: text, path: text, currency: text, state: text });
const member = shape<OrgMember>({ member: text, name: text, targets: array(shape({ id: text, name: text })),
  history: array(shape({ target: text, name: text, from: text, until: text })) });
export const organizationsPageOf = shape<OrgPage>({ revision: text, relationship_revision: integer, active_revision: integer, on: text,
  workspace: shape({ id: text, profile: text }), parties: array(party), profiles: array(profile),
  kind: choice("hierarchy", "payer", "contract_pool", "allocation_pool", "view_group"), members: array(member), next: text,
  filters: shape({ subject: text, entity: text, unattributed: boolean, q: text, member_count: integer, target: text }),
  provider_filter: text, scan_stopped: boolean, scan_budget: integer,
});
const node = shape<OrgTreeNode>({ id: text, name: text, kind: choice("legal_entity", "internal_unit", "account", "unknown"), state: text,
  source: choice("relationship", "registry", "none"), children: choice("some", "none", "unchecked") });
export const organizationTreeOf = shape<OrgTreePage>({ revision: text, relationship_revision: integer, active_revision: integer, on: text, parent: text, group: text,
  nodes: array(node), next: text, scan_stopped: boolean, registry_empty: boolean });
