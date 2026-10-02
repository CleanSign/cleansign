import { MigrationConfig } from "drizzle-orm/migrator";
import { NodePgDatabase } from "drizzle-orm/node-postgres";
import { migrate } from "drizzle-orm/node-postgres/migrator";

const migrationConfig: MigrationConfig = {
  migrationsFolder: "./migrations",
};

export async function runMigrations(
  db: NodePgDatabase<Record<string, unknown>>
) {
  await migrate(db, migrationConfig);
}
