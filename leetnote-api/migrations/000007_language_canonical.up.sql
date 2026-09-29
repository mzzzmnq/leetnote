-- =============================================================
-- 000007_language_canonical · 收敛解法语言到规范值
-- =============================================================
--
-- 背景：solutions.language 原本是自由文本（VARCHAR(20)，无约束），
-- 意味着 Python / python / PY / py 会存成四个不同的值。
-- 一旦前端要「按语言分组」或者统计"我用几种语言写过解法"，
-- 这种脏数据就会让结果完全错乱，而且事后很难清洗。
--
-- 本迁移做两件事：
--   1. 把历史数据归一化（别名 + 大小写 → 规范形式）
--   2. 加 CHECK 约束兜住未来 —— 应用层已经会归一化，这里是第二道防线
--
-- 规范值定义在 internal/model/language.go 的 SupportedLanguages，
-- 两边必须一致；改动其一要同时改另一个。
-- =============================================================

-- ---------------------------------------------------------------
-- 1. 历史数据归一化
--
-- 对已经是规范值的行是幂等的（ELSE 分支会让它保持原样），
-- 所以本语句可以安全重复执行。
-- ---------------------------------------------------------------
UPDATE solutions
SET language = CASE lower(trim(language))
    -- Python
    WHEN 'py'        THEN 'python'
    WHEN 'python2'   THEN 'python'
    WHEN 'python3'   THEN 'python'
    WHEN 'python'    THEN 'python'
    -- Go
    WHEN 'golang'    THEN 'go'
    WHEN 'go'        THEN 'go'
    -- JavaScript
    WHEN 'js'        THEN 'javascript'
    WHEN 'node'      THEN 'javascript'
    WHEN 'nodejs'    THEN 'javascript'
    WHEN 'javascript' THEN 'javascript'
    -- TypeScript
    WHEN 'ts'        THEN 'typescript'
    WHEN 'typescript' THEN 'typescript'
    -- Java
    WHEN 'java'      THEN 'java'
    -- C++
    WHEN 'c++'       THEN 'cpp'
    WHEN 'cxx'       THEN 'cpp'
    WHEN 'cplusplus' THEN 'cpp'
    WHEN 'cpp'       THEN 'cpp'
    -- C
    WHEN 'gcc'       THEN 'c'
    WHEN 'c'         THEN 'c'
    ELSE lower(trim(language))
END;

-- ---------------------------------------------------------------
-- 2. 归一化之后仍然有"不认识"的语言就直接报错
--
-- 【为什么不静默改掉】削足适履地把 rust 改成 python 会悄悄丢信息。
-- 宁可让迁移失败，把选择权交给执行的人：要么补进白名单，要么手工改数据。
-- ---------------------------------------------------------------
DO $$
DECLARE
    unsupported text;
    allowed     text := 'python, go, typescript, javascript, java, cpp, c';
BEGIN
    SELECT string_agg(DISTINCT quote_literal(language), ', ' ORDER BY quote_literal(language))
      INTO unsupported
      FROM solutions
     WHERE language NOT IN ('python', 'go', 'typescript', 'javascript', 'java', 'cpp', 'c');

    IF unsupported IS NOT NULL THEN
        RAISE EXCEPTION
            E'存在不受支持的语言：%\n请在 internal/model/language.go 的 SupportedLanguages 里补上（并同步本文件与前端），或先把这些数据改掉。\n当前允许值：%',
            unsupported, allowed;
    END IF;
END
$$;

-- ---------------------------------------------------------------
-- 3. 加约束
-- ---------------------------------------------------------------
ALTER TABLE solutions
    ADD CONSTRAINT solutions_language_check
    CHECK (language IN ('python', 'go', 'typescript', 'javascript', 'java', 'cpp', 'c'));

COMMENT ON COLUMN solutions.language IS
    '编程语言，规范的小写形式。可选值见 internal/model/language.go 的 SupportedLanguages';
