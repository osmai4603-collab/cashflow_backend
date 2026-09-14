# معيار أخطاء RFC 7807 (Problem Details for HTTP APIs)

يُعد معيار **RFC 7807** هو المعيار الدولي القياسي لتمثيل الأخطاء في واجهات RESTful الحديثة، مما يتيح للعملاء (Clients) معالجة الأخطاء برمجياً بشكل متسق دون الاعتماد على تحليل نصوص عشوائية.

---

## بنية الخطأ المعياري

عند حدوث أي خطأ، يُعاد الرأس:
`Content-Type: application/problem+json`

وحمولة الخطأ:

```json
{
  "type": "https://api.cashflow.com/errors/insufficient-funds",
  "title": "Insufficient Funds",
  "status": 422,
  "detail": "Account 987 has balance 200.00, cannot withdraw 500.00",
  "instance": "/api/v1/accounts/987/withdraw",
  "code": "ACCOUNT_INSUFFICIENT_FUNDS",
  "invalid_params": [
    {
      "name": "amount",
      "reason": "must be less than or equal to current balance"
    }
  ]
}
```

---

## الحقول الأساسية

- **`type`**: رابط URI يوثق نوع الخطأ وسببه المعماري.
- **`title`**: ملخص بشري قصير وموجز للخطأ (ثابت لنفس النوع).
- **`status`**: رمز حالة HTTP المتطابق مع ترويسة الاستجابة.
- **`detail`**: شرح تفصيلي عن الحالة المحددة التي أدت للخطأ في هذا الطلب.
- **`code`**: رمز كودي مخصص للآلة (Machine-readable) يسهل ربطه بشاشات الواجهة الأمامية والترجمة.
- **`invalid_params`**: مصفوفة تفصيلية بحقول الإدخال التي فشلت في التحقق.
