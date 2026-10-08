import { pgTable, text, timestamp, boolean, index } from "drizzle-orm/pg-core";
import { user } from "./auth";

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
