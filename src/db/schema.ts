import {
  pgTable,
  pgEnum,
  text,
  timestamp,
  boolean,
  integer,
  real,
  jsonb,
  index,
} from "drizzle-orm/pg-core";
import { desc } from "drizzle-orm";
import { user } from "./auth-schema";

export * from "./auth-schema";

export const contractStatus = pgEnum("contract_status", [
  "pending",
  "completed",
  "failed",
]);

export const clauseSeverity = pgEnum("clause_severity", [
  "high",
  "medium",
  "low",
]);

export const userConsent = pgTable(
  "user_consent",
  {
    id: text("id")
      .primaryKey()
      .$defaultFn(() => crypto.randomUUID()),
    userId: text("user_id")
      .notNull()
      .references(() => user.id, { onDelete: "cascade" }),
    terms: boolean("terms").notNull(),
    privacy: boolean("privacy").notNull(),
    marketing: boolean("marketing").notNull().default(false),
    createdAt: timestamp("created_at")
      .$defaultFn(() => new Date())
      .notNull(),
  },
  (table) => [index("user_consent_user_id_idx").on(table.userId)],
);

export const contract = pgTable(
  "contract",
  {
    id: text("id")
      .primaryKey()
      .$defaultFn(() => crypto.randomUUID()),
    userId: text("user_id")
      .notNull()
      .references(() => user.id, { onDelete: "cascade" }),
    fileName: text("file_name").notNull(),
    // 오브젝트 스토리지 연결 전까지는 원본을 저장하지 않으므로 null
    fileKey: text("file_key"),
    mimeType: text("mime_type").notNull(),
    status: contractStatus("status").notNull().default("pending"),
    lossAmount: integer("loss_amount").notNull().default(0),
    riskScore: integer("risk_score"),
    grade: clauseSeverity("grade"),
    lossMin: integer("loss_min"),
    lossMax: integer("loss_max"),
    depositAmount: integer("deposit_amount"),
    modelVersion: text("model_version"),
    warnings: jsonb("warnings").$type<string[]>(),
    riskFlow: jsonb("risk_flow"),
    // 분석 중(pending)일 때만 의미가 있다. stage: reading | splitting | classifying | scoring
    progress: jsonb("progress").$type<{ stage: string; done: number; total: number }>(),
    errorMessage: text("error_message"),
    createdAt: timestamp("created_at")
      .$defaultFn(() => new Date())
      .notNull(),
  },
  (table) => [
    index("contract_user_id_created_at_idx").on(
      table.userId,
      desc(table.createdAt),
    ),
  ],
);

export const contractDangerClause = pgTable(
  "contract_danger_clause",
  {
    id: text("id")
      .primaryKey()
      .$defaultFn(() => crypto.randomUUID()),
    contractId: text("contract_id")
      .notNull()
      .references(() => contract.id, { onDelete: "cascade" }),
    original: text("original").notNull(),
    clauseType: text("clause_type"),
    confidence: real("confidence"),
    lMax: integer("l_max"),
    lossAmount: integer("loss_amount").notNull().default(0),
    severity: clauseSeverity("severity").notNull(),
    reason: text("reason").notNull(),
    lawReference: text("law_reference"),
    safeText: text("safe_text").notNull(),
    orderIndex: integer("order_index").notNull().default(0),
    createdAt: timestamp("created_at")
      .$defaultFn(() => new Date())
      .notNull(),
  },
  (table) => [
    index("contract_danger_clause_contract_id_idx").on(table.contractId),
  ],
);
