/**
 * Whether the result screen shows the overall contract risk score and grade.
 * Keep it off until "Accurate contract risk grade" (#31) is done.
 * Server-only and read per request, so it can be switched with the deployment env without a rebuild.
 */
export const riskScoreEnabled = () => process.env.SHOW_RISK_SCORE === "true";
