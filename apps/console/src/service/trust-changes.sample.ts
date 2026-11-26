export const trustWire = {
 "list": {
  "rows": [
   {
    "id": "trust-18fac15f-7",
    "number": 7,
    "state": "independently_approved",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3300,
     "step_up": []
    },
    "proposer": "sample-admin",
    "validator": "sample-admin-2",
    "approver": "sample-admin-2",
    "activator": "",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "validated",
    "evidence_current": "not_evaluated",
    "invalidation_state": "",
    "decide_in_detail": true,
    "can": {
     "revise": true,
     "validate": true,
     "reject": false,
     "cancel": true
    },
    "why_not": {
     "approve": "self",
     "activate": "self"
    },
    "act_until": "2026-11-16T09:10:00Z"
   },
   {
    "id": "trust-18fac15f-6",
    "number": 6,
    "state": "validated",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3240,
     "step_up": []
    },
    "proposer": "sample-admin-2",
    "validator": "sample-admin-3",
    "approver": "",
    "activator": "",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "validated",
    "evidence_current": "not_evaluated",
    "invalidation_state": "",
    "decide_in_detail": true,
    "can": {
     "revise": false,
     "validate": true,
     "reject": true,
     "cancel": false
    },
    "why_not": {
     "approve": "",
     "activate": "needs_approval"
    },
    "act_until": "2026-11-16T09:10:00Z"
   },
   {
    "id": "trust-18fac15f-5",
    "number": 5,
    "state": "draft",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3180,
     "step_up": []
    },
    "proposer": "sample-admin",
    "validator": "",
    "approver": "",
    "activator": "",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "",
    "evidence_current": "not_applicable",
    "invalidation_state": "",
    "decide_in_detail": true,
    "can": {
     "revise": true,
     "validate": true,
     "reject": false,
     "cancel": true
    },
    "why_not": {
     "approve": "self",
     "activate": "self"
    },
    "act_until": "2026-11-16T09:10:00Z"
   }
  ],
  "search_input": "",
  "query": {
   "q": "",
   "state": "",
   "limit": 3
  },
  "next": "cursor-1",
  "reference_at": "2026-11-16T09:00:00Z",
  "observed_at": "2026-11-16T09:00:00Z",
  "revision": "r11",
  "order": "created_desc",
  "count": {
   "value": "7",
   "certainty": "exact"
  },
  "work_units": 3,
  "scan_stopped": false
 },
 "open": {
  "rows": [
   {
    "id": "trust-18fac15f-7",
    "number": 7,
    "state": "independently_approved",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3300,
     "step_up": []
    },
    "proposer": "sample-admin",
    "validator": "sample-admin-2",
    "approver": "sample-admin-2",
    "activator": "",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "validated",
    "evidence_current": "not_evaluated",
    "invalidation_state": "",
    "decide_in_detail": true,
    "can": {
     "revise": false,
     "validate": true,
     "reject": true,
     "cancel": false
    },
    "why_not": {
     "approve": "not_validated",
     "activate": "approver"
    },
    "act_until": "2026-11-16T09:10:00Z"
   },
   {
    "id": "trust-18fac15f-6",
    "number": 6,
    "state": "validated",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3240,
     "step_up": []
    },
    "proposer": "sample-admin-2",
    "validator": "sample-admin-3",
    "approver": "",
    "activator": "",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "validated",
    "evidence_current": "not_evaluated",
    "invalidation_state": "",
    "decide_in_detail": true,
    "can": {
     "revise": true,
     "validate": true,
     "reject": false,
     "cancel": true
    },
    "why_not": {
     "approve": "self",
     "activate": "self"
    },
    "act_until": "2026-11-16T09:10:00Z"
   },
   {
    "id": "trust-18fac15f-5",
    "number": 5,
    "state": "draft",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3180,
     "step_up": []
    },
    "proposer": "sample-admin",
    "validator": "",
    "approver": "",
    "activator": "",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "",
    "evidence_current": "not_applicable",
    "invalidation_state": "",
    "decide_in_detail": true,
    "can": {
     "revise": false,
     "validate": true,
     "reject": true,
     "cancel": false
    },
    "why_not": {
     "approve": "not_validated",
     "activate": "needs_approval"
    },
    "act_until": "2026-11-16T09:10:00Z"
   }
  ],
  "search_input": "TRUST-",
  "query": {
   "q": "trust-",
   "state": "open",
   "limit": 25
  },
  "next": "",
  "reference_at": "2026-11-16T09:00:00Z",
  "observed_at": "2026-11-16T09:00:00Z",
  "revision": "r11",
  "order": "created_desc",
  "count": {
   "value": "3",
   "certainty": "exact"
  },
  "work_units": 3,
  "scan_stopped": false
 },
 "detail": {
  "change": {
   "id": "trust-18fac15f-7",
   "number": 7,
   "state": "independently_approved",
   "change_revision": 1,
   "change_digest": "47b6910996269e776d83fe6a0194bb4f418fd790ada3b9b50da7a97f0c8ce012",
   "old_epoch": {
    "value": "2",
    "generation": 0,
    "number": 2
   },
   "old_policy": 2,
   "old_digest": "af179a8e130d4fcde866be8626e1614ac50a6e1c122cd678dbd666a15fb54b02",
   "new_digest": "4b444ef5d3d9c284a1f7a5434185aa9090de5190614945ba14e8b0f89685233e",
   "candidate": {
    "methods": [
     "password"
    ],
    "mfa": "none",
    "max_age_seconds": 3300,
    "step_up": []
   },
   "proposer": "sample-admin",
   "validator": "sample-admin-2",
   "approver": "sample-admin-2",
   "activator": "",
   "superseded_by": "",
   "superseded_epoch": null,
   "earlier_run": false,
   "run_started_at": "",
   "validation": {
    "result": "validated",
    "at": "2026-11-16T09:00:00Z",
    "evidence_digest": "7d5b24a00e3b1b542bc604b5d8b0a8bf8e108d5e5e587a816d413830979df17d",
    "evidence_current": true,
    "items": [
     {
      "item": "test_login",
      "result": "verified",
      "reason": "",
      "detail": ""
     },
     {
      "item": "recovery_path",
      "result": "verified",
      "reason": "",
      "detail": ""
     },
     {
      "item": "issuer_subject",
      "result": "not_applicable",
      "reason": "local_password",
      "detail": ""
     },
     {
      "item": "provider_diagnostics",
      "result": "not_applicable",
      "reason": "no_provider_connection",
      "detail": ""
     },
     {
      "item": "external_status",
      "result": "not_applicable",
      "reason": "no_external_identity",
      "detail": ""
     },
     {
      "item": "mfa_weakening",
      "result": "information",
      "reason": "unchanged",
      "detail": ""
     },
     {
      "item": "affected_subjects",
      "result": "information",
      "reason": "",
      "detail": "3"
     },
     {
      "item": "session_impact",
      "result": "information",
      "reason": "",
      "detail": "3"
     },
     {
      "item": "record_size",
      "result": "verified",
      "reason": "",
      "detail": ""
     }
    ]
   },
   "impact": {
    "members": 3,
    "sessions": 3,
    "affected_ids": [
     "sample-admin",
     "sample-admin-2",
     "sample-admin-3"
    ],
    "weakens_mfa": false,
    "signs_out_everyone": true
   },
   "activation": null,
   "job": null,
   "invalidation": null,
   "can": {
    "revise": true,
    "validate": true,
    "approve": false,
    "activate": false,
    "reject": false,
    "cancel": true
   },
   "why_not": {
    "approve": "self",
    "activate": "self"
   },
   "act_until": "2026-11-16T09:10:00Z",
   "history": [
    {
     "state": "draft",
     "by": "sample-admin",
     "at": "2026-11-16T09:00:00Z",
     "rationale": "sample reason"
    },
    {
     "state": "validated",
     "by": "sample-admin-2",
     "at": "2026-11-16T09:00:00Z",
     "rationale": "sample reason"
    },
    {
     "state": "independently_approved",
     "by": "sample-admin-2",
     "at": "2026-11-16T09:00:00Z",
     "rationale": "sample reason"
    }
   ]
  },
  "revision": "r11",
  "observed_at": "2026-11-16T09:00:00Z"
 },
 "held": {
  "rows": [
   {
    "id": "trust-1050a94a-1",
    "number": 1,
    "state": "activation_unknown",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3600,
     "step_up": []
    },
    "proposer": "sample-admin",
    "validator": "sample-admin-2",
    "approver": "sample-admin-2",
    "activator": "sample-admin-3",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "validated",
    "evidence_current": "not_applicable",
    "invalidation_state": "",
    "decide_in_detail": false,
    "can": {
     "revise": false,
     "validate": false,
     "reject": false,
     "cancel": false
    },
    "why_not": {
     "approve": "trust_unavailable",
     "activate": "trust_unavailable"
    },
    "act_until": "2026-11-16T09:15:00Z"
   }
  ],
  "search_input": "",
  "query": {
   "q": "",
   "state": "",
   "limit": 25
  },
  "next": "",
  "reference_at": "2026-11-16T09:00:00Z",
  "observed_at": "2026-11-16T09:00:00Z",
  "revision": "r11",
  "order": "created_desc",
  "count": {
   "value": "1",
   "certainty": "exact"
  },
  "work_units": 1,
  "scan_stopped": false
 },
 "closed": {
  "rows": [
   {
    "id": "trust-11111111-1",
    "number": 0,
    "state": "activated",
    "change_revision": 0,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3300,
     "step_up": []
    },
    "proposer": "",
    "validator": "",
    "approver": "",
    "activator": "",
    "superseded_by": "",
    "earlier_run": true,
    "run_started_at": "2026-11-16T09:00:00Z",
    "validation_result": "",
    "evidence_current": "not_applicable",
    "invalidation_state": "",
    "decide_in_detail": false,
    "can": {
     "revise": false,
     "validate": false,
     "reject": false,
     "cancel": false
    },
    "why_not": {
     "approve": "not_validated",
     "activate": "needs_approval"
    },
    "act_until": "2026-11-16T09:10:00Z"
   },
   {
    "id": "trust-d31eef46-4",
    "number": 4,
    "state": "activated",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3600,
     "step_up": [
      {
       "action": "policy.change",
       "max_age_seconds": 600,
       "mfa": false
      }
     ]
    },
    "proposer": "sample-admin",
    "validator": "sample-admin-2",
    "approver": "sample-admin-2",
    "activator": "sample-admin-3",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "validated",
    "evidence_current": "not_applicable",
    "invalidation_state": "complete",
    "decide_in_detail": false,
    "can": {
     "revise": false,
     "validate": false,
     "reject": false,
     "cancel": false
    },
    "why_not": {
     "approve": "self",
     "activate": "self"
    },
    "act_until": "2026-11-16T09:10:00Z"
   },
   {
    "id": "trust-d31eef46-3",
    "number": 3,
    "state": "superseded",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3120,
     "step_up": []
    },
    "proposer": "sample-admin-2",
    "validator": "",
    "approver": "",
    "activator": "",
    "superseded_by": "trust-d31eef46-4",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "",
    "evidence_current": "not_applicable",
    "invalidation_state": "",
    "decide_in_detail": false,
    "can": {
     "revise": false,
     "validate": false,
     "reject": false,
     "cancel": false
    },
    "why_not": {
     "approve": "not_validated",
     "activate": "needs_approval"
    },
    "act_until": "2026-11-16T09:10:00Z"
   },
   {
    "id": "trust-d31eef46-2",
    "number": 2,
    "state": "rejected",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3060,
     "step_up": []
    },
    "proposer": "sample-admin",
    "validator": "",
    "approver": "",
    "activator": "",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "",
    "evidence_current": "not_applicable",
    "invalidation_state": "",
    "decide_in_detail": false,
    "can": {
     "revise": false,
     "validate": false,
     "reject": false,
     "cancel": false
    },
    "why_not": {
     "approve": "self",
     "activate": "self"
    },
    "act_until": "2026-11-16T09:10:00Z"
   },
   {
    "id": "trust-d31eef46-1",
    "number": 1,
    "state": "cancelled",
    "change_revision": 1,
    "candidate": {
     "methods": [
      "password"
     ],
     "mfa": "none",
     "max_age_seconds": 3000,
     "step_up": []
    },
    "proposer": "sample-admin",
    "validator": "",
    "approver": "",
    "activator": "",
    "superseded_by": "",
    "earlier_run": false,
    "run_started_at": "",
    "validation_result": "",
    "evidence_current": "not_applicable",
    "invalidation_state": "",
    "decide_in_detail": false,
    "can": {
     "revise": false,
     "validate": false,
     "reject": false,
     "cancel": false
    },
    "why_not": {
     "approve": "self",
     "activate": "self"
    },
    "act_until": "2026-11-16T09:10:00Z"
   }
  ],
  "search_input": "",
  "query": {
   "q": "",
   "state": "closed",
   "limit": 25
  },
  "next": "",
  "reference_at": "2026-11-16T09:00:00Z",
  "observed_at": "2026-11-16T09:00:00Z",
  "revision": "r11",
  "order": "created_desc",
  "count": {
   "value": "5",
   "certainty": "exact"
  },
  "work_units": 5,
  "scan_stopped": false
 }
} as const;
