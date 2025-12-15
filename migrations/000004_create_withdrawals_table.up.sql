create table withdrawals
(
    user_id      integer not null,
    order_number bigint not null,
    sum          integer not null,
    constraint withdrawals_pk primary key (user_id, order_number)
);