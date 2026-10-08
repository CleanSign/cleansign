import { betterAuth } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import { db } from "@/db/db";
import * as schema from "@/db/schema/auth";
import { nextCookies } from "better-auth/next-js";
import { emailOTP } from "better-auth/plugins";
import { sendEmail } from "@/lib/email-client";
import OtpEmail from "@/email/otp";

const smtpConfigured =
  !!process.env.SMTP_HOST && process.env.SMTP_HOST !== "smtp.example.com";

export const auth = betterAuth({
  database: drizzleAdapter(db, {
    provider: "pg",
    schema: schema,
  }),
  plugins: [
    nextCookies(),
    emailOTP({
      async sendVerificationOTP({ email, otp, type }) {
        if (!smtpConfigured) {
          console.log("✉️ [dev] OTP for", email, ":", otp, "(type:", type, ")");
          return;
        }
        await sendEmail(email, "Clean Sign 인증코드", OtpEmail({ otp }));
      },
    }),
  ],
});
