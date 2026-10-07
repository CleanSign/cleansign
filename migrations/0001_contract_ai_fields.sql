ALTER TABLE "contract" ALTER COLUMN "file_key" DROP NOT NULL;--> statement-breakpoint
ALTER TABLE "contract" ADD COLUMN "grade" "clause_severity";--> statement-breakpoint
ALTER TABLE "contract" ADD COLUMN "loss_min" integer;--> statement-breakpoint
ALTER TABLE "contract" ADD COLUMN "loss_max" integer;--> statement-breakpoint
ALTER TABLE "contract" ADD COLUMN "deposit_amount" integer;--> statement-breakpoint
ALTER TABLE "contract" ADD COLUMN "model_version" text;--> statement-breakpoint
ALTER TABLE "contract" ADD COLUMN "warnings" jsonb;--> statement-breakpoint
ALTER TABLE "contract_danger_clause" ADD COLUMN "clause_type" text;--> statement-breakpoint
ALTER TABLE "contract_danger_clause" ADD COLUMN "confidence" real;--> statement-breakpoint
ALTER TABLE "contract_danger_clause" ADD COLUMN "l_max" integer;