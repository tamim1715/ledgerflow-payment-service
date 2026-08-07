-- 000003_seed_demo_customers.down.sql

DELETE FROM customers WHERE id IN ('cust_987654', 'cust_12345');
