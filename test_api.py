import requests

HOST = '127.0.0.1'
PORT = '3000'
URL = 'http://' + HOST + ':' + PORT


def test_handles():
    r = requests.get(URL + '/' + "get_balance", params={'user_id': '3'})
    res = r.json()
    assert 'Response' in res and res['Response'] == 'there is no rows with given user_id'

    r = requests.get(URL + '/' + "get_balance", params={'user_id': '4'})
    res = r.json()
    assert 'Response' in res and res['Response'] == 'there is no rows with given user_id'

    r = requests.get(URL + '/' + "top_up_balance", params={'user_id': '3', 'accrued_amount': '100'})
    res = r.json()
    assert 'Response' in res and res['Response'] == 'ok'

    r = requests.get(URL + '/' + "write_off_money", params={'user_id': '3', 'debited_amount': '1'})
    res = r.json()
    assert 'Response' in res and res['Response'] == 'ok'

    r = requests.get(URL + '/' + "transfer_money", params={'from_user_id': '3', 'to_user_id': '4', 'amount': '66'})
    res = r.json()
    assert 'Response' in res and res['Response'] == 'ok'

    r = requests.get(URL + '/' + "get_balance", params={'user_id': '4'})
    res = r.json()
    assert 'Users' in res and len(res['Users']) == 1 and res['Users'][0]['UserId'] == 4 and res['Users'][0][
        'Amount'] == 66

    r = requests.get(URL + '/' + "get_balance", params={'user_id': '3'})
    res = r.json()
    assert 'Users' in res and len(res['Users']) == 1 and res['Users'][0]['UserId'] == 3 and res['Users'][0][
        'Amount'] == 33

    r = requests.get(URL + '/' + "get_user_history", params={'user_id': '3', 'n_last_operations': '5'})
    res = r.json()
    assert 'Histories' in res
    res = res['Histories']
    tmp = [{'UserId': 3, 'IsDebit': True, 'Amount': 66},
           {'UserId': 3, 'IsDebit': True, 'Amount': 1},
           {'UserId': 3, 'IsDebit': False, 'Amount': 100}]
    for i, val in enumerate(tmp):
        assert val['UserId'] == res[i]['UserId']
        assert val['IsDebit'] == res[i]['IsDebit']
        assert val['Amount'] == res[i]['Amount']


if __name__ == '__main__':
    test_handles()
