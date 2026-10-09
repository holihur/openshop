/** Roles that may operate the console, least privileged first. */
export const OPS_ROLES = ["support", "catalog", "finance", "admin"] as const;

export type OpsRole = (typeof OPS_ROLES)[number];

/**
 * The roles an operator may assign. It mirrors the API's validation so the
 * console cannot offer a role the server would reject.
 */
export function appropriateRoles(): OpsRole[] {
  return [...OPS_ROLES];
}

/** Human description of a role, used where a badge needs a tooltip. */
export const ROLE_SUMMARY: Record<OpsRole, string> = {
  support: "support",
  catalog: "catalog",
  finance: "finance",
  admin: "admin",
};
