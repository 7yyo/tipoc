WITH RECURSIVE fibonacci (n, fib_n, next_fib_n) AS
                   (SELECT 1, 0, 1
                    UNION ALL
                    SELECT n + 1, next_fib_n, fib_n + next_fib_n
                    FROM fibonacci
                    WHERE n < 10)
SELECT * FROM fibonacci;