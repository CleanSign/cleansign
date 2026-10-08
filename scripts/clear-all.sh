# This script clears the migrations folder and the database.
# It is useful for resetting the development environment.

rm -rf migrations/*
echo "✅ Migrations folder cleared."

# Remove domain schemas, keep only the auth schema
rm -f src/db/schema/user.ts src/db/schema/contract.ts
echo "✅ Domain schema files cleared."

# Reset the schema index to export only the auth schema
echo 'export * from "./auth";' > src/db/schema/index.ts
echo "✅ Schema index reset."

# Print that everything is now ready
echo "✅ All done! Your development environment is ready."
echo "📝 Don't forget to run 'npm run create:migration' to create a new migration after setting up your 'src/db/schema'."
echo "📝 Run 'npm run create:auth:migration' if any changes were done to 'src/lib/auth.ts'!"
echo "📝 Make sure your '.env' file is set so that the application can work as intended."
