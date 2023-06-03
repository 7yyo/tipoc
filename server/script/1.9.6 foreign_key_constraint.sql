CREATE TABLE ${TABLE_NAME}_customers (customer_id INT PRIMARY KEY, customer_name VARCHAR(11));
CREATE TABLE ${TABLE_NAME}_orders ( order_id INT PRIMARY KEY, customer_id INT, order_date DATE,
                        FOREIGN KEY (customer_id) REFERENCES ${TABLE_NAME}_customers(customer_id));
INSERT INTO ${TABLE_NAME}_customers VALUES (1, 'Alice'), (2, 'Bob'), (3, 'Charlie');
INSERT INTO ${TABLE_NAME}_orders VALUES (1, 1, '2023-05-30'), (2, 2, '2023-05-30'), (3, 4, '2023-05-30');
