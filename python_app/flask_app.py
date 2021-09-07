#!/usr/bin/env python

from sqlalchemy import create_engine
from sqlalchemy import Column, Integer, String
from sqlalchemy import Integer, String, Column
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from flask import session, request, Flask, jsonify
import time

Base = declarative_base()
app = Flask(__name__)
app.config['SECRET_KEY'] = 'test_random_key' # ?


class Users(Base):
    __tablename__ = 'users'
    user_id = Column(Integer, primary_key=True)
    amount = Column(Integer)


@app.route('/get_user_info', methods=['GET'])
def get_user_info():
    if request.method == 'GET':
        user_id = request.args.get('user_id', 0, type=int)
        db_session = app.config['db_session']
        u = db_session().query(Users).where(Users.user_id == user_id).first()
        if u:
            return jsonify(user_id=u.user_id, amount=u.amount)
        else:
            return f'There is no user {user_id}', 404
    # curl -X GET "http://127.0.0.1:8192/get_user_info?user_id=3"


@app.route('/top_up_balance', methods=['GET'])
def top_up_balance():
    if request.method == 'GET':
        user_id = request.args.get('user_id', 0, type=int)
        update_amount = request.args.get('update_amount', 0, type=int)
        if update_amount < 0:
            return 'amount should be >= 0', 500
        db_session = app.config['db_session']
        with db_session.begin() as trans_session:
            try:
                u = trans_session.query(Users).where(Users.user_id == user_id).with_for_update().one_or_none()
                if u:
                    u.amount += update_amount
                else:
                    new_user = Users(user_id=user_id, amount=update_amount)
                    trans_session.add(new_user)
            except Exception as e:
                return "problems in transaction", 500
        return 'ok', 200
    # curl -X GET "http://127.0.0.1:8192/top_up_balance?user_id=3&update_amount=333"

    # psql -U postgres
    # \c account_keeper
    # update users set amount=111 where user_id=3;


@app.route('/write_off_money', methods=['GET'])
def write_off_money():
    if request.method == 'GET':
        from_user_id = request.args.get('from_user_id', 0, type=int)
        to_user_id = request.args.get('to_user_id', 0, type=int)
        amount = request.args.get('amount', 0, type=int)
        if amount < 0:
            return 'amount should be >= 0', 500
        db_session = app.config['db_session']
        with db_session.begin() as trans_session:
            try:
                from_u = trans_session.query(Users).where(Users.user_id == from_user_id).with_for_update().one_or_none()
                to_u = trans_session.query(Users).where(Users.user_id == to_user_id).with_for_update().one_or_none()
                if not from_u:
                    return f"there is no user_id {from_user_id}"
                if from_u.amount < amount:
                    return 'not enough money', 500
                if to_u:
                    from_u.amount -= amount
                    to_u.amount += amount
                else:
                    from_u.amount -= amount
                    new_user = Users(user_id=to_user_id, amount=amount)
                    trans_session.add(new_user)
            except Exception as e:
                return "problems in transaction", 500
        return 'ok', 200
    # curl -X GET "http://127.0.0.1:8192/write_off_money?from_user_id=3&to_user_id=4&amount=10"


if __name__ == '__main__':
    postgres_db = {'username': 'postgres',
                   'password': 'postgres',
                   'host': '0.0.0.0',
                   'port': 5432,
                   'db': 'account_keeper'}

    engine = create_engine('postgresql://{}:{}@{}:{}/{}'.format(
        postgres_db['username'],
        postgres_db['password'],
        postgres_db['host'],
        postgres_db['port'],
        postgres_db['db']), echo=True)
    session_maker = sessionmaker(bind=engine)
    # session = session_maker()
    app.config['db_session'] = session_maker
    app.run(host="0.0.0.0", port=8192)
