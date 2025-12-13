create table orders
(
    number     bigint not null constraint orders_pk primary key,
    user_id    integer constraint orders_users_id_fk references users,
    status     varchar(16) not null,
    created_at TIMESTAMP default CURRENT_TIMESTAMP,
    updated_at TIMESTAMP default CURRENT_TIMESTAMP,
    accrual    integer
);