/* Open a terminal and run the following commands to run the go file:
 * export GOPATH=$HOME/go
 * export GOBIN=$GOPATH/bin
 * go get
 * go run server.go ...
 * 
 * Open another terminal and type the following command to start mongodb:
 * mongod
 *
 * Open a third terminal window and type in the commands written under 'Usage'
 * for each method.
 */ 

package main

//Import statements
import(
  "fmt"
  "log"
  "net/http"
  "encoding/json"
  "strconv"

  "github.com/gorilla/mux"    //For router
  "gopkg.in/mgo.v2"           //For database
  "gopkg.in/mgo.v2/bson"
)

//Student struct
type Student struct{
  NetID   string  `json:"netid"bson:"netid"`
  Name    string  `json:"name"bson:"name"`
  Major   string  `json:"major"bson:"major"`
  Year    int     `json:"year"bson:"year"`
  Grade   int     `json:"grade"bson:"grade"`
  Rating  string  `json:"rating"bson:"rating"`
}

var (
   session *mgo.Session
   coll *mgo.Collection
   Err error
)

//Main function
func main(){

  session, Err = mgo.Dial("localhost")
    if (Err != nil){
      fmt.Println("Error connecting to Mongo. Please make sure you are following all steps and retry.\n")
      return
    }
  session.SetMode(mgo.Monotonic, true)
  coll = session.DB("test").C("db")
  if ( coll != nil ) {
    fmt.Println("Got a collection object.")
  }


  router := mux.NewRouter()
  router.Methods("GET").Path("/Student/getstudent").Handler(http.HandlerFunc(get))
  router.Methods("GET").Path("/Student/listall").Handler(http.HandlerFunc(list))
  router.Methods("POST").Path("/Student").Handler(http.HandlerFunc(post)) 
  router.Methods("DELETE").Path("/Student/{value}").Handler(http.HandlerFunc(delete))
  router.Methods("PATCH").Path("/Student").Handler(http.HandlerFunc(update))

  log.Fatal(http.ListenAndServe(":1234", router)) //listen on port 1234
}
 
//GET the student based on identifier
//Usage: curl -L -X GET http://localhost:1234/Student/getstudent?name=Mike
func get(w http.ResponseWriter, r *http.Request){
  var result Student
  keys := r.URL.Query()

  for k,_ := range keys{
    err := coll.Find(bson.M{k: keys[k][0]}).One(&result)
    if (err == mgo.ErrNotFound) {
      fmt.Fprintf(w, "No user found \n")
    } else if (err != nil) {
      panic(err)
      fmt.Fprintf(w, "Panic while finding the student. Breaking out.\n")
    } else {
      response, err := json.MarshalIndent(result, "", "  ")
      if err != nil{
        fmt.Fprintf(w, "Panic while marshalling. Breaking out.\n")
        return
      }
      fmt.Fprintf(w, string(response))
      fmt.Fprintf(w, "\n")
    }
  }
}

//GET list all info for all student
//Usage: curl -L -X GET http://localhost:1234/Student/listall
func list(w http.ResponseWriter, r *http.Request){
  var results []Student

  err := coll.Find(nil).All(&results)
  if err != nil{
    fmt.Fprintf(w, "Panic while finding students. Breaking out.\n")
    return
  }
  
  //MarshalResults
  for s,_ := range results{
    response, err := json.MarshalIndent(results[s], "", "  ")
    if err != nil{
      fmt.Fprintf(w, "Panic while Marshalling. Breaking out.\n")
      return
    }
    fmt.Fprintf(w, "Student\n")
    fmt.Fprintf(w, string(response))
    fmt.Fprintf(w, "\n")
  }
}

//POST adds the user to the database, if the id is unique
//Usage: curl -L -X POST -d '{"NetID":"147001234", "Name":"Mike","Major":"CS","Year":2015,"Grade":90,"Rating":"D"}' http://localhost:1234/Student
func post(w http.ResponseWriter, r *http.Request){
  decoder := json.NewDecoder(r.Body)
  var s,match Student

  err := decoder.Decode(&s)
  if err != nil{
    fmt.Fprintf(w, "Panic while Decoding. Breaking out.\n")  
    return
  }

  err = coll.Find(bson.M{"netid": s.NetID}).One(&match)
  if err == mgo.ErrNotFound{
    err = coll.Insert(&s)
    if err != nil{
      fmt.Fprintf(w, "Panic while Inserting. Breaking out.\n")
      return
    }
    fmt.Fprintf(w, "Added user\n")
  } else if err != nil {
    fmt.Fprintf(w,"Error while searching for similer student. Breaking out\n")
    return
  } else {
    fmt.Fprintf(w, "User with the same netid already exists\n")
  }
}    

//DELETE deletes all users with year <= the passed in value
//Usage: curl -L -X DELETE http://localhost:1234/Student/2018
func delete(w http.ResponseWriter, r *http.Request){
  year, err := strconv.ParseInt(mux.Vars(r)["value"],0,0)
  if err != nil{
    fmt.Fprintf(w, "Error parsing year to an int. Breaking out.\n")
    return
  }

  info, err := coll.RemoveAll(bson.M{"year": bson.M{"$lte": year}})
  if err != nil{
    fmt.Fprintf(w, "Panic while trying to remove students. Breaking out.\n")  
    return
  }
  fmt.Fprintf(w, "%d student[s] removed!\n", info.Removed)
}

//PATCH updates grades for all users in the db based on the avg computed
//Usage: curl -L -X PATCH http://localhost:1234/Student
func update(w http.ResponseWriter, r *http.Request){
  var results []Student
  avg := 0
  size := 0 

  err := coll.Find(nil).All(&results)
  if err != nil{
    fmt.Fprintf(w, "Error while trying to find students. Breaking out.\n")
    return
  }

  for s,_ := range results{
    avg = avg + results[s].Grade
    size++
  }

  if size != 0 {
    avg = avg/size
  } else {
    fmt.Fprintf(w, "No students in database. Braking out.\n")
    return
  }

  for s,_ := range results{
    if (results[s].Grade >= avg + 10) {
      err = coll.Update(bson.M{"netid": results[s].NetID}, bson.M{"$set": bson.M{"rating": "A"}})
      if err != nil {
        fmt.Fprintf(w, "Updating error. Breaking out.\n")
        return
      }
    } else if (results[s].Grade >= avg - 10) && (results[s].Grade < avg + 10) {
      err = coll.Update(bson.M{"netid": results[s].NetID}, bson.M{"$set": bson.M{"rating": "B"}})
      if err != nil {
        fmt.Fprintf(w, "Updating error. Breaking out.\n")
        return
      }
    } else if (results[s].Grade >= avg - 20) && (results[s].Grade < avg - 10) {
      err = coll.Update(bson.M{"netid": results[s].NetID}, bson.M{"$set": bson.M{"rating": "C"}})
      if err != nil {
        fmt.Fprintf(w, "Updating error. Breaking out.\n")
        return
      }
    } 
  }
  fmt.Fprintf(w, "Average was %v.\nUpdated information:\n", avg)
  list(w,r)
}
