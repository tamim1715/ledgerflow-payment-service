-- 000003_seed_demo_customers.up.sql

INSERT INTO customers (id, email, name, phone)
VALUES 
    ('cust_987654', 'john.doe@example.com', 'John Doe', '+1234567890'),
    ('cust_12345', 'jane.smith@example.com', 'Jane Smith', '+0987654321')
ON CONFLICT (id) DO NOTHING;
