CREATE TABLE ${TABLE_NAME} (id INT PRIMARY KEY, c1 INT);
INSERT INTO ${TABLE_NAME} VALUES (1, 1);
ALTER TABLE ${TABLE_NAME} ADD INDEX k (`c1`);
ANALYZE TABLE ${TABLE_NAME};
SELECT * FROM ${TABLE_NAME} WHERE c1 = 1;
SELECT /*+ IGNORE_INDEX(${TABLE_NAME}, k) */ * FROM ${TABLE_NAME} WHERE c1 = 1;
SELECT SLEEP(3);
SELECT query_sample_text, digest, plan FROM information_schema.statements_summary
WHERE digest = (SELECT digest FROM information_schema.statements_summary
                WHERE query_sample_text = 'SELECT * FROM ${TABLE_NAME} WHERE c1 = 1'
                   OR query_sample_text = 'SELECT /*+ ignore_index(${TABLE_NAME}, k) */ * FROM ${TABLE_NAME} WHERE c1 = 1'
                GROUP BY digest);