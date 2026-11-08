import { stages, type Closing, type Stage, type Status } from "./model";
export type ClosingFilter = {
  query: string;
  stage: Stage | "";
  status: Status | "";
};
export const emptyClosingFilter: ClosingFilter = {
  query: "",
  stage: "",
  status: "",
};
export function filterClosings(rows: Closing[], filter: ClosingFilter) {
  const query = filter.query.trim().toLowerCase();
  return rows.filter(
    (row) =>
      row.organization.toLowerCase().includes(query) &&
      (!filter.status ||
        (filter.stage
          ? row.status[filter.stage] === filter.status
          : stages.some((stage) => row.status[stage] === filter.status))),
  );
}
