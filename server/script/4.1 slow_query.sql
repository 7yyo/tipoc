CREATE TABLE ${TABLE_NAME} (id INT, c1 INT, c2 INT, name VARCHAR(11));
INSERT INTO ${TABLE_NAME} VALUES (1, 10, 100, 'Green');
SET tidb_slow_log_threshold = 0;
SELECT * FROM ${TABLE_NAME} WHERE name = 'Green';
SELECT time, query, query_time, user, host
FROM information_schema.slow_query
WHERE query = 'SELECT * FROM ${TABLE_NAME} WHERE name = ''Green'';'
ORDER BY time desc LIMIT 1;
