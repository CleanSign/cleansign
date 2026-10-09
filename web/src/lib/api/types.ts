import type { components } from "./schema";

// server/api/openapi.yaml에서 생성한 타입 (npm run gen:api)
export type Contract = components["schemas"]["Contract"];
export type RiskItem = components["schemas"]["RiskItem"];
export type ClauseDetail = components["schemas"]["ClauseDetail"];
export type ContractListItem = components["schemas"]["ContractListItem"];
export type ContractPage = components["schemas"]["ContractPage"];
export type Progress = components["schemas"]["Progress"];
export type Me = components["schemas"]["Me"];
export type Tokens = components["schemas"]["Tokens"];
