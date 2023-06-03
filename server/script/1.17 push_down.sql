CREATE TABLE ${TABLE_NAME} (id INT, name VARCHAR(11), age INT);
INSERT INTO ${TABLE_NAME} VALUE (1, 'Green', 18),(2, 'Jim', 24);
EXPLAIN ANALYZE SELECT COUNT(*) FROM ${TABLE_NAME};
INSERT INTO mysql.expr_pushdown_blacklist VALUES('COUNT','tikv','');
ADMIN RELOAD expr_pushdown_blacklist;
EXPLAIN ANALYZE SELECT COUNT(*) FROM ${TABLE_NAME};
DELETE FROM mysql.expr_pushdown_blacklist WHERE name = 'COUNT';
ADMIN RELOAD expr_pushdown_blacklist;
EXPLAIN ANALYZE SELECT COUNT(*) FROM ${TABLE_NAME};
