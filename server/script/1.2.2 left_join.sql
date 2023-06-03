CREATE TABLE ${TABLE_NAME}_user (id INT PRIMARY KEY , name VARCHAR(11));
CREATE TABLE ${TABLE_NAME}_salary (id INT PRIMARY KEY, user_id INT, salary DECIMAL(10,2));
INSERT INTO ${TABLE_NAME}_user VALUES (1, 'Jim'), (2, 'Green'), (3, 'Ming');
INSERT INTO ${TABLE_NAME}_salary VALUE (1, 1, 188.88), (2, 2, 299.99);
SELECT u.name, s.salary FROM ${TABLE_NAME}_user u LEFT JOIN ${TABLE_NAME}_salary s ON u.id = s.user_id;