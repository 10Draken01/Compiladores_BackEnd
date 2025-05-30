from pymongo import MongoClient

def obtener_usuarios_paginacion(page=1, limit=100):
    client = MongoClient("mongodb://localhost:27017")
    db = client.lexicodb
    collection = db.users

    skip = (page - 1) * limit

    cursor = collection.find(
        {},  # filtro vacío, trae todos
        sort=[("Clave_Cliente", 1)]  # orden ascendente por Clave_Cliente
    ).skip(skip).limit(limit)

    resultados = list(cursor)
    return resultados

if __name__ == "__main__":
    page = 1
    usuarios = obtener_usuarios_paginacion(page=page, limit=100)
    for usuario in usuarios:
        print(usuario)
