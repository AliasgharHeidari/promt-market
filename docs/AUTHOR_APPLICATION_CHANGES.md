# تغییرات: فلوی ارتقا به Prompt Author

## فایل‌های جدید (کپی کن سر جاشون)
```
internal/domain/author_application.go          [جدید]
internal/repository/author_application_repository.go [جدید]
internal/service/author_application_service.go [جدید]
internal/handler/author_application_handler.go [جدید]
internal/middleware/prompt_author_only.go       [جدید]
migrations/0001_author_application.up.sql       [جدید]
```

## فایل‌های جایگزین (این‌ها کامل بازنویسی شدن، فایل قبلی رو overwrite کن)
```
internal/domain/user.go            → فیلدهای Phone, PhoneVerified, NationalID اضافه شد
internal/repository/verification_repository.go → متدهای OTP موبایل اضافه شد
internal/service/admin_service.go  → پارامتر appService به NewAdminService اضافه شد + متدهای بررسی درخواست
internal/handler/admin_handler.go  → NewAdminHandler حالا redisClient هم می‌گیره + endpointهای جدید
internal/routes/routes.go          → روت‌های /author-application اضافه شد + PromptAuthorOnly روی ساخت پرامپت
internal/routes/admin_routes.go    → NewAdminHandler صدا زدنش عوض شد + روت‌های بررسی درخواست
```

## ⚠️ نکته مهم: امضای NewAdminHandler عوض شده
قبل: `handler.NewAdminHandler(db)`
بعد: `handler.NewAdminHandler(db, redisClient)`

اگه جای دیگه‌ای (مثلاً تست) صداش می‌زنی، باید آپدیت کنی.

## اجرای Migration
```bash
psql -U <user> -d <dbname> -f migrations/0001_author_application.up.sql
```
یا اگه از GORM AutoMigrate استفاده می‌کنی، مطمئن شو `domain.AuthorApplication{}` و `domain.User{}`
توی لیست AutoMigrate هست (توی `internal/database/postgres.go`).

## پوشه آپلود
عکس کارت ملی/شناسنامه توی `./uploads/id_documents/<user_id>/` ذخیره می‌شه (نسبت به working directory برنامه).
مطمئن شو این پوشه در `.gitignore` و در `docker-compose.yml` به‌صورت volume mount شده باشه که با ری‌استارت کانتینر پاک نشه:

```yaml
services:
  api:
    volumes:
      - ./uploads:/app/uploads
```

## فلوی تست کامل (curl)

```bash
# 1. ثبت‌نام و تایید ایمیل (فرض: از قبل انجام شده و لاگین کردی)
TOKEN="<access_token>"

# 2. درخواست کد OTP موبایل — کد توی ترمینال سرور چاپ می‌شه
curl -X POST http://localhost:PORT/api/v1/author-application/send-phone-otp \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"phone": "09121234567"}'

# لاگ سرور رو ببین:
# 📱 [SMS SIMULATION] OTP for user <id> (phone 09121234567): 483920 (expires in 5m0s)

# 3. تایید کد
curl -X POST http://localhost:PORT/api/v1/author-application/verify-phone \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code": "483920"}'

# 4. ارسال درخواست (multipart/form-data)
curl -X POST http://localhost:PORT/api/v1/author-application \
  -H "Authorization: Bearer $TOKEN" \
  -F "expertise=Prompt Engineering, Marketing" \
  -F "national_id=1234567890" \
  -F "id_document=@/path/to/id_card.jpg"

# 5. چک وضعیت درخواست خودت
curl http://localhost:PORT/api/v1/author-application/me \
  -H "Authorization: Bearer $TOKEN"

# 6. (به عنوان ادمین) دیدن درخواست‌های pending
curl http://localhost:PORT/api/v1/admin/author-applications?status=pending \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 7. (ادمین) دیدن عکس مدرک
curl http://localhost:PORT/api/v1/admin/author-applications/<app_id>/document \
  -H "Authorization: Bearer $ADMIN_TOKEN" --output doc.jpg

# 8. (ادمین) تایید
curl -X PUT http://localhost:PORT/api/v1/admin/author-applications/<app_id>/approve \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# حالا Role کاربر "prompt_author" شده و می‌تونه پرامپت بسازه:
curl -X POST http://localhost:PORT/api/v1/prompts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{...}'
```

## نکات باقی‌مونده برای بعد (به‌صورت عمدی ساده نگه داشته شدن)
- **SMS واقعی:** توی `author_application_service.go` تابع `SendPhoneOTP`، خط
  `log.Printf("📱 [SMS SIMULATION]...")` رو با فراخوانی SDK سرویس SMS جایگزین کن.
- **اعتبارسنجی کد ملی:** الان فقط چک می‌کنه ۱۰ رقمیه. اگه چک‌سام رسمی کد ملی ایران
  رو هم می‌خوای، تابع `validateNationalID` رو گسترش بده.
- **محدودیت یک اپلیکیشن در هر لحظه:** اگه یه کاربر رد بشه، می‌تونه دوباره apply کنه
  (چون status قبلی‌ش rejected هست). اگه pending یا approved باشه، اجازه apply مجدد نمی‌ده.