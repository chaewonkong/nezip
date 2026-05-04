CREATE TABLE IF NOT EXISTS prices (
    apt_name TEXT    NOT NULL,
    lawd_cd  TEXT    NOT NULL,
    month    TEXT    NOT NULL,  -- YYYYMM
    area_int INTEGER NOT NULL,  -- ROUND(excluUseAr)
    area     REAL    NOT NULL,  -- 원본 면적
    median   INTEGER NOT NULL,
    count    INTEGER NOT NULL,
    PRIMARY KEY (apt_name, lawd_cd, area_int, month)
);
