CREATE TABLE ${TABLE_NAME}_A (
                            id INT PRIMARY KEY,
                            name VARCHAR(50),
                            age INT,
                            grade VARCHAR(10)
);
CREATE TABLE ${TABLE_NAME}_B (
                            id INT PRIMARY KEY,
                            name VARCHAR(50),
                            age INT,
                            grade VARCHAR(10)
);
INSERT INTO ${TABLE_NAME}_A (id, name, age, grade) VALUES
                                                  (1, 'Alice', 18, 'A'),
                                                  (2, 'Bob', 17, 'B'),
                                                  (3, 'Charlie', 19, 'A');
INSERT INTO ${TABLE_NAME}_B (id, name, age, grade) VALUES
                                                  (4, 'David', 20, 'C'),
                                                  (5, 'Emma', 18, 'A'),
                                                  (6, 'Frank', 17, 'B');
SELECT id, name, age, grade FROM ${TABLE_NAME}_A
UNION
SELECT id, name, age, grade FROM ${TABLE_NAME}_B;
