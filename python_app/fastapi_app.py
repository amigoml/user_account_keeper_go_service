#!/usr/bin/env python

from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import declarative_base, sessionmaker
from sqlalchemy import Column, Integer
from sqlalchemy.future import select
from sqlalchemy import text

from fastapi import APIRouter, FastAPI
from pydantic import BaseModel

app = FastAPI()
router = APIRouter()
postgres_db = {'username': 'postgres',
               'password': 'postgres',
               'host': '0.0.0.0',
               'port': 5432,
               'db': 'account_keeper'}

DB_URL = 'postgresql+asyncpg://{}:{}@{}:{}/{}'.format(
    postgres_db['username'],
    postgres_db['password'],
    postgres_db['host'],
    postgres_db['port'],
    postgres_db['db'])

engine = create_async_engine(DB_URL, future=True, echo=True)
async_session = sessionmaker(engine, expire_on_commit=False, class_=AsyncSession)
Base = declarative_base()


class Users(Base):
    __tablename__ = 'users'
    user_id = Column(Integer, primary_key=True)
    amount = Column(Integer)


class UserResp(BaseModel):
    userId: int
    amount: int


@app.get('/get_user_info')
async def get_user_info(user_id: int, response_model=UserResp):
    from sqlalchemy.future import select
    async with async_session() as session:
        async with session.begin():
            q = await session.execute(select(Users).where(Users.user_id == user_id))
            u = q.scalars().first()
            if u:
                return UserResp(userId=u.user_id, amount=u.amount)
            else:
                return f'There is no user {user_id}'
# curl -X GET "http://127.0.0.1:8192/get_user_info?user_id=3"


@app.get('/top_up_balance')
async def top_up_balance(user_id: int, update_amount: int):
    async with async_session() as session:
        async with session.begin():
            q = await session.execute(
                text(f"select user_id, amount from users where user_id = {user_id} for update")
            )
            u = q.one_or_none()
            if u:
                res = await session.execute(
                    text(f"update users set amount = amount + {update_amount} where user_id = {user_id}")
                )
                return 'ok'
            else:
                res = await session.execute(
                    text(f"insert into users (user_id, amount) values ({user_id}, {update_amount})")
                )
    return "ok"
# curl -X GET "http://127.0.0.1:8192/top_up_balance?user_id=3&update_amount=2"


@app.get('/write_off_money')
async def write_off_money(user_id: int, debited_amount: int):
    if debited_amount < 0:
        return "debited_amount should be greater than 0"
    async with async_session() as session:
        async with session.begin():
            q = await session.execute(
                text(f"select user_id, amount from users where user_id = {user_id} for update")
            )
            u = q.one_or_none()
            if u is None:
                return f'There is no user {user_id}'
            if u.amount < debited_amount:
                return f'Not enough money'
            res = await session.execute(
                text(f"update users set amount = amount - {debited_amount} where user_id = {user_id}")
            )
            print(res)
    return 'ok'
# curl -X GET "http://127.0.0.1:8192/write_off_money?user_id=3&debited_amount=2"


@app.get('/transfer_money')
async def transfer_money(from_user_id: int, to_user_id: int, amount: int):
    async with async_session() as session:
        async with session.begin():
            q = await session.execute(
                text(f"select user_id, amount from users where user_id = {from_user_id} for update")
            )
            u = q.one_or_none()
            if u is None:
                return 'err'
            print(type(u), u)
            if u.amount < amount:
                return 'not enough money to transfer them'

            q1 = await session.execute(
                text(f"select user_id, amount from users where user_id = {to_user_id} for update")
            )
            u1 = q1.one_or_none()
            is_new_user = False
            if u1 is None:
                is_new_user = True

            res = await session.execute(
                text(f"update users set amount = amount - {amount} where user_id = {from_user_id}")
            )
            if is_new_user:
                res = await session.execute(
                    text(f"insert into users (user_id, amount) values ({to_user_id}, {amount})")
                )
            else:
                res = await session.execute(
                    text(f"update users set amount = amount + {amount} where user_id = {to_user_id}")
                )

    return 'ok'
#  curl -X GET "http://127.0.0.1:8192/transfer_money?from_user_id=3&to_user_id=4&amount=10"

# не совсем понятно все ли ошибки я обрабатываю + нет обратки входных значений

# uvicorn fastapi_app:app --reload --port 8192
# psql -U postgres
# \c account_keeper
