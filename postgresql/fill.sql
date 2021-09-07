-- Set params
set session my.number_of_users = '10';
set session my.max_amount = '1000';

-- load the pgcrypto extension to gen_random_uuid ()
CREATE EXTENSION pgcrypto;

-- Filling of users
INSERT INTO users
select id
	, floor(random() * (current_setting('my.max_amount')::int))::int
FROM GENERATE_SERIES(1, current_setting('my.number_of_users')::int) as id;
