create table withdrawals
(
    user_id      integer not null,
    order_number bigint not null,
    sum          integer not null,
    processed_at TIMESTAMP default CURRENT_TIMESTAMP,
    constraint withdrawals_pk primary key (user_id, order_number)
);