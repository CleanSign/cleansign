import { Html, Text, Heading } from "@react-email/components";

export default function OtpEmail({ otp }: { otp: string }) {
  return (
    <Html>
      <Heading as="h2">Clean Sign 인증코드</Heading>
      <Text>아래 인증코드를 입력해 로그인을 완료하세요.</Text>
      <Text style={{ fontSize: "28px", fontWeight: 700, letterSpacing: "4px" }}>
        {otp}
      </Text>
      <Text>이 코드는 잠시 후 만료됩니다. 본인이 요청하지 않았다면 이 메일을 무시하세요.</Text>
    </Html>
  );
}
