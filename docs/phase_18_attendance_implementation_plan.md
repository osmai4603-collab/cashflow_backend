# المرحلة 18: نظام الحضور والانصراف (HR Attendance) - خطة التنفيذ التفصيلية

تستهدف هذه المرحلة بناء نظام حضور وانصراف متكامل يعتمد على معايير Odoo 19.0، مع دعم متقدم للعمل الإضافي والتتبع الجغرافي.

## 1. هيكل البيانات (Domain Entities)

### Attendance
```go
type Attendance struct {
    ID             int64
    EmployeeID     int64
    CheckIn        time.Time
    CheckOut       *time.Time
    WorkedHours    float64    // محسوب: CheckOut - CheckIn (مع خصم فترات الراحة)
    ExpectedHours  float64    // الساعات المقررة بناءً على التقويم
    OvertimeHours  float64    // إجمالي ساعات الإضافي المرتبطة
    OvertimeStatus string     // to_approve, approved, refused
    
    // التتبع الجغرافي والتقني
    InLatitude     float64
    InLongitude    float64
    InIPAddress    string
    InBrowser      string
    InMode         string     // kiosk, systray, manual, technical
    
    OutLatitude    float64
    OutLongitude   float64
    OutIPAddress   string
    OutBrowser     string
    OutMode        string     // kiosk, systray, manual, technical, auto_check_out
    
    CompanyID      int64
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### Overtime Management
```go
type OvertimeLine struct {
    ID             int64
    EmployeeID     int64
    Date           time.Time
    Duration       float64    // الساعات الإضافية المحسوبة
    ManualDuration float64    // الساعات المعتمدة يدوياً
    Status         string     // to_approve, approved, refused
    TimeStart      time.Time
    TimeStop       time.Time
    RuleIDs        []int64    // القواعد المطبقة
}

type OvertimeRule struct {
    ID             int64
    Name           string
    BaseOff        string     // quantity (عدد ساعات), timing (توقيت محدد)
    TimingType     string     // work_days, non_work_days, leave, schedule
    TimingStart    float64
    Multiplier     float64    // معامل الاحتساب (1.5, 2.0)
}
```

## 2. ميزات Odoo 19.0 المضافة
- **Tolerance (التسامح):** دعم `overtime_company_threshold` و `overtime_employee_threshold` (بالدقائق) لتجنب احتساب فوارق زمنية بسيطة.
- **Kiosk Mode (وضع الكشك):** واجهة مخصصة لتسجيل الحضور عبر PIN أو Barcode مع دعم `attendance_kiosk_delay`.
- **Auto Check-out:** إمكانية الخروج التلقائي للموظفين الذين نسوا تسجيل الخروج مع `auto_check_out_tolerance`.

## 3. واجهات API
- `POST /api/v1/attendance/check-in`: تسجيل دخول مع إرسال بيانات الموقع.
- `POST /api/v1/attendance/check-out`: تسجيل خروج.
- `GET /api/v1/attendance/kiosk-config`: جلب إعدادات وضع الكشك.
- `POST /api/v1/overtime/approve`: اعتماد ساعات الإضافي (للمديرين).
- `GET /api/v1/employees/{id}/attendance-report`: تقرير الحضور التفصيلي.

## 4. التكامل
- **الموارد البشرية (HR):** ربط سجلات الحضور بملفات الموظفين والتقاويم الخاصة بهم.
- **الأنشطة:** تنبيه المديرين عند وجود ساعات إضافي تحتاج لاعتماد.
- **الرواتب (المستقبل):** تصدير ساعات العمل المعتمدة لمعالجة كشوف المرتبات.

## 5. خطة التحقق
- اختبار سيناريو تسجيل الدخول والخروج وحساب الساعات بدقة.
- اختبار قواعد الإضافي (يوم عمل عادي مقابل عطلة نهاية أسبوع).
- التحقق من منع التداخل بين فترات الحضور لنفس الموظف.
