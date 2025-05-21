# agregar a 1000 usuarios a la base de datos a la api port 8000 localhost /api/users { "Clave_Cliente": int, "Nombre": str, "Celular": str, "Email": str }
# usaremos faker para generar datos aleatorios
import requests
import json
from faker import Faker

fake = Faker()
# url de la api
url = 'http://localhost:8000/api/users'
# headers de la api
headers = {
    'Content-Type': 'application/json'
}
# datos de la api
data = {
    "Clave_Cliente": 0,
    "Nombre": "",
    "Celular": "",
    "Email": ""
}
# funcion para agregar usuarios
def agregar_usuarios():
    for i in range(1,100001):
        # generar datos aleatorios
        data['Clave_Cliente'] = i
        data['Nombre'] = fake.name()
        data['Celular'] = fake.phone_number()
        data['Email'] = fake.email()
        # convertir datos a json
        json_data = json.dumps(data)
        # hacer peticion post a la api
        response = requests.post(url, headers=headers, data=json_data)
        # imprimir respuesta
        print(response.status_code)
        print(response.json())
        

agregar_usuarios()