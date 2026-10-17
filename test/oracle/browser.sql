WHENEVER OSERROR EXIT FAILURE
WHENEVER SQLERROR EXIT SQL.SQLCODE ROLLBACK
SET DEFINE OFF

ALTER SESSION SET CONTAINER = FREEPDB1;

-- A materialized view's owner needs CREATE TABLE directly, not only via DBA.
GRANT CREATE TABLE TO lazyora;

-- Idempotent fixture for schema browsing; existing sample rows are preserved.
CREATE OR REPLACE VIEW lazyora.sample_rows_view AS
    SELECT id, title, amount FROM lazyora.sample_rows;

COMMENT ON COLUMN lazyora.sample_rows.id IS '식별자 / Identifier';
COMMENT ON COLUMN lazyora.sample_rows_view.title IS '뷰 제목 / View title';

CREATE OR REPLACE FUNCTION lazyora.browser_function(p_value IN NUMBER DEFAULT 1) RETURN NUMBER IS
BEGIN
    RETURN p_value;
END;
/

CREATE OR REPLACE PROCEDURE lazyora.browser_procedure(p_in IN NUMBER, p_out OUT NUMBER) IS
BEGIN
    p_out := p_in;
END;
/

CREATE OR REPLACE PACKAGE lazyora.browser_package AS
    PROCEDURE ping;
    FUNCTION echo_value(p_value IN NUMBER) RETURN NUMBER;
    FUNCTION echo_value(p_value IN VARCHAR2) RETURN VARCHAR2;
END;
/
CREATE OR REPLACE PACKAGE BODY lazyora.browser_package AS
    PROCEDURE ping IS BEGIN NULL; END;
    FUNCTION echo_value(p_value IN NUMBER) RETURN NUMBER IS BEGIN RETURN p_value; END;
    FUNCTION echo_value(p_value IN VARCHAR2) RETURN VARCHAR2 IS BEGIN RETURN p_value; END;
END;
/

CREATE OR REPLACE TYPE lazyora.browser_type AS OBJECT (
    n NUMBER,
    MEMBER FUNCTION value_number RETURN NUMBER
);
/
CREATE OR REPLACE TYPE BODY lazyora.browser_type AS
    MEMBER FUNCTION value_number RETURN NUMBER IS
    BEGIN
        RETURN self.n;
    END;
END;
/

CREATE OR REPLACE SYNONYM lazyora.browser_synonym FOR lazyora.sample_rows;

CREATE OR REPLACE TRIGGER lazyora.browser_trigger
BEFORE UPDATE ON lazyora.sample_rows
FOR EACH ROW
BEGIN
    NULL;
END;
/

-- These object kinds have no CREATE OR REPLACE form on older Oracle versions.
BEGIN
    EXECUTE IMMEDIATE 'CREATE SEQUENCE lazyora.browser_sequence';
EXCEPTION WHEN OTHERS THEN
    IF SQLCODE != -955 THEN RAISE; END IF;
END;
/
BEGIN
    EXECUTE IMMEDIATE 'CREATE INDEX lazyora.browser_index ON lazyora.sample_rows (amount)';
EXCEPTION WHEN OTHERS THEN
    IF SQLCODE != -955 THEN RAISE; END IF;
END;
/
BEGIN
    EXECUTE IMMEDIATE 'CREATE MATERIALIZED VIEW lazyora.browser_mview BUILD IMMEDIATE REFRESH COMPLETE ON DEMAND AS SELECT id, title FROM lazyora.sample_rows';
EXCEPTION WHEN OTHERS THEN
    IF SQLCODE != -12006 AND SQLCODE != -955 THEN RAISE; END IF;
END;
/

-- Replacing dependent objects above can invalidate an existing materialized view.
ALTER MATERIALIZED VIEW lazyora.browser_mview COMPILE;

COMMENT ON COLUMN lazyora.browser_mview.title IS '구체화 뷰 제목 / Materialized view title';
