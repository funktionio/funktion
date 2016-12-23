## Blog Example

This example shows how to combine some functions and flows.

```json

funktion create function -n blogcount -f examples/blog-example/blogcount.js
funktion create function -n split -f examples/blog-example/split.js

funktion subscribe -n blogendpoint -c http4 http://localhost http://blogcount

export SPLIT=`minikube service --url split -n funky`
echo $SPLIT

curl -X POST --header "Content-Type: application/json"  -d '
[
  {
    "userId": 1,
    "id": 1,
    "title": "sunt aut facere repellat provident occaecati excepturi optio reprehenderit",
    "body": "quia et suscipit\nsuscipit recusandae consequuntur expedita et cum\nreprehenderit molestiae ut ut quas totam\nnostrum rerum est autem sunt rem eveniet architecto"
  },
  {
    "userId": 1,
    "id": 2,
    "title": "qui est esse",
    "body": "est rerum tempore vitae\nsequi sint nihil reprehenderit dolor beatae ea dolores neque\nfugiat blanditiis voluptate porro vel nihil molestiae ut reiciendis\nqui aperiam non debitis possimus qui neque nisi nulla"
  }
]
' $SPLIT
```

