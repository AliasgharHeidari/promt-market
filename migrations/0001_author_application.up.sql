-- افزودن فیلدهای شماره موبایل و کد ملی به جدول users
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone           VARCHAR(20)  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phone_verified  BOOLEAN      NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS national_id     VARCHAR(20)  NOT NULL DEFAULT '';

-- جدول درخواست‌های ارتقا به prompt_author
CREATE TABLE IF NOT EXISTS author_applications (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expertise         VARCHAR(150) NOT NULL,
    national_id       VARCHAR(20)  NOT NULL,
    id_document_path  VARCHAR(500) NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'pending', -- pending | approved | rejected
    rejection_reason  TEXT,
    reviewed_by       UUID REFERENCES users(id),
    reviewed_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_author_applications_user_id ON author_applications(user_id);
CREATE INDEX IF NOT EXISTS idx_author_applications_status  ON author_applications(status);