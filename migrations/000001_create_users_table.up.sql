create table users
(
    id       serial not null constraint id primary key,
    login    varchar(255) not null constraint login_unique unique,
    password varchar(64)
);