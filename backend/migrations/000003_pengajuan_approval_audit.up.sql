-- Migrasi 000003: Tabel Pengajuan, Keputusan Approval, dan Audit Trail
-- Menunjang FR-08 (Approval Berjenjang) dan FR-09 (Audit Trail).

CREATE TABLE IF NOT EXISTS pengajuan (
    id               BIGSERIAL PRIMARY KEY,
    nomor_referensi  VARCHAR(32) UNIQUE NOT NULL,
    total_plafon     BIGINT NOT NULL CHECK (total_plafon >= 0),
    grade            SMALLINT CHECK (grade BETWEEN 1 AND 5),
    status           VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
    created_by       BIGINT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pengajuan_status ON pengajuan(status);
CREATE INDEX IF NOT EXISTS idx_pengajuan_created_by ON pengajuan(created_by);

-- Keputusan approval per level (FR-08, BR-02, BR-10)
CREATE TABLE IF NOT EXISTS keputusan_approval (
    id           BIGSERIAL PRIMARY KEY,
    pengajuan_id BIGINT NOT NULL REFERENCES pengajuan(id) ON DELETE RESTRICT,
    level        VARCHAR(8) NOT NULL,
    keputusan    VARCHAR(16) NOT NULL CHECK (keputusan IN ('APPROVE', 'REJECT', 'RETURN')),
    alasan       TEXT,
    catatan      TEXT,
    approver_id  BIGINT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_keputusan_approval_pengajuan ON keputusan_approval(pengajuan_id);

-- Audit trail append-only (FR-09, BR-10, BR-11, AC-13)
-- Tabel ini tidak memiliki constraint ON DELETE CASCADE, tidak boleh dimutasi
CREATE TABLE IF NOT EXISTS audit_trail (
    id             BIGSERIAL PRIMARY KEY,
    pengajuan_id   BIGINT REFERENCES pengajuan(id) ON DELETE RESTRICT,
    aksi           VARCHAR(64) NOT NULL,
    status_sebelum VARCHAR(32),
    status_sesudah VARCHAR(32),
    catatan        TEXT,
    actor_id       BIGINT NOT NULL,
    actor_role     VARCHAR(16) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_trail_pengajuan ON audit_trail(pengajuan_id, created_at ASC);
