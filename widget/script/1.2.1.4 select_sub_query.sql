CREATE TABLE ${TABLE_NAME}(order_id INT PRIMARY KEY,customer_id INT,order_date DATE,total_amount DECIMAL(10, 2));
INSERT INTO ${TABLE_NAME} (order_id, customer_id, order_date, total_amount) VALUES
       (1, 101, '2023-07-15', 120.50),
       (2, 102, '2023-07-16', 85.75),
       (3, 101, '2023-07-17', 65.20),
       (4, 103, '2023-07-17', 240.90),
       (5, 102, '2023-07-18', 50.00);

SELECT customer_id,
       (SELECT COUNT(*) FROM ${TABLE_NAME} o WHERE o.customer_id = c.customer_id) AS order_count
FROM (SELECT DISTINCT customer_id FROM ${TABLE_NAME}) c;
