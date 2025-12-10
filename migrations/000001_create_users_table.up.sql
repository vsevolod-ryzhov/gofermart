create table users
(
    login    varchar(255) not null constraint login primary key,
    password varchar(64)
);