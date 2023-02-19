package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type Article struct {
	Id                        uint16
	Title, Anons, ArticleText string
}

/*
 * Необходимо для отображения всех статей на главное странице
 * Данный список наполняется и передается в шаблон в качестве аргумента
 */
var articles = []Article{}

/*
 * Необходимо для отображения конкретной статьи полностью
 * Заполняется и передается в шаблон
 */
var tmpArticle = Article{}

func main() {
	handleRequest()
}

func handleRequest() {
	router := mux.NewRouter()
	router.HandleFunc("/", index).Methods("GET")
	router.HandleFunc("/create", create).Methods("GET")
	router.HandleFunc("/edit/{id:[0-9]+}", edit).Methods("GET")
	router.HandleFunc("/article/{id:[0-9]+}", getArticleById).Methods("GET")
	router.HandleFunc("/warning", warning).Methods("GET")

	router.HandleFunc("/save_article", save).Methods("POST")
	router.HandleFunc("/update/{id:[0-9]+}", update).Methods("POST")
	router.HandleFunc("/delete/{id:[0-9]+}", delete).Methods("GET", "DELETE")

	http.Handle("/", router)
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}

	getAllArticles()
	t.ExecuteTemplate(w, "index", articles)
}

/*
 * Отображение страницы для создания новой статьи
 */
func create(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/create.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}

	t.ExecuteTemplate(w, "create", nil)
}

/*
 * Сохранение статъи в базе данных
 * Достаем значения из формы по наименования поля в шаблоне html
 */
func save(w http.ResponseWriter, r *http.Request) {
	titleValue := r.FormValue("title")
	anonsValue := r.FormValue("anons")
	articleValue := r.FormValue("article_text")

	if isEmpty(titleValue, anonsValue, articleValue) {
		warning(w, r)
	} else {
		db := connectToDb()
		createArticle(db, titleValue, anonsValue, articleValue)
		defer db.Close()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func isEmpty(title string, anons string, article string) bool {
	return title == "" || anons == "" || article == ""
}

/*
 * Обновление запсиси в таблице
 */
func update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	titleValue := r.FormValue("title")
	anonsValue := r.FormValue("anons")
	articleValue := r.FormValue("article_text")

	if isEmpty(titleValue, anonsValue, articleValue) {
		warning(w, r)
	} else {
		db := connectToDb()
		put, err := db.Query(fmt.Sprintf("UPDATE `articles` SET `title` = '%s', `anons` = '%s', `article_text` = '%s' WHERE `id` = '%s'",
			titleValue, anonsValue, articleValue, vars["id"]))
		if err != nil {
			panic(err)
		}

		defer put.Close()
		defer db.Close()

		http.Redirect(w, r, fmt.Sprintf("/article/%s", vars["id"]), http.StatusSeeOther)
	}
}

/*
 * Удаление статьи
 */
func delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	db := connectToDb()
	res, err := db.Query(fmt.Sprintf("DELETE FROM `articles` WHERE `id` = '%s'", vars["id"]))

	tmpArticle = Article{}
	for res.Next() {
		var article Article
		err = res.Scan(&article.Id, &article.Title, &article.Anons, &article.ArticleText)
		if err != nil {
			panic(err)
		}

		tmpArticle = article
	}

	defer db.Close()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

/*
 * Полное отображение статьи по идентификатору (при нажатии кнопки "Read more")
 */
func getArticleById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t, err := template.ParseFiles("templates/show.html", "templates/header.html", "templates/footer.html")
	db := connectToDb()
	res, err := db.Query(fmt.Sprintf("SELECT * FROM `articles` WHERE `id` = '%s'", vars["id"]))

	tmpArticle = Article{}
	for res.Next() {
		var article Article
		err = res.Scan(&article.Id, &article.Title, &article.Anons, &article.ArticleText)
		if err != nil {
			panic(err)
		}

		tmpArticle = article
	}

	t.ExecuteTemplate(w, "show", tmpArticle)
}

/*
 * Получение всех статей из базы данных
 * Выполняем запрос, полученные данные формируем в список
 */
func getAllArticles() {
	db := connectToDb()
	res, err := db.Query("SELECT * FROM `articles`")
	if err != nil {
		panic(err)
	}

	articles = []Article{}
	for res.Next() {
		var article Article
		err = res.Scan(&article.Id, &article.Title, &article.Anons, &article.ArticleText)
		if err != nil {
			panic(err)
		}

		articles = append(articles, article)
	}

	defer db.Close()
}

/*
 * Настройка подключения к БД
 * Укажите свои параметры для подключения
 */
func connectToDb() *sql.DB {
	const username = "{$USERNAME}"
	const password = "{$PASSWORD}"
	const database = "{$DATABASE}"

	connection := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s", username, password, database)
	db, err := sql.Open("mysql", connection)
	if err != nil {
		panic(err.Error())
	}

	return db
}

func createArticle(db *sql.DB, title string, anons string, article string) {
	insert, err := db.Query(fmt.Sprintf(
		"INSERT INTO `articles` (`title`, `anons`, `article_text`) VALUES('%s', '%s', '%s')",
		title, anons, article))

	if err != nil {
		panic(err.Error())
	}
	defer insert.Close()
}

/*
 * Форма редактирования статьи по идентификатору
 */
func edit(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/edit.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}

	vars := mux.Vars(r)
	db := connectToDb()
	res, err := db.Query(fmt.Sprintf("SELECT * FROM `articles` WHERE `id` = '%s'", vars["id"]))

	tmpArticle = Article{}
	for res.Next() {
		var article Article
		err = res.Scan(&article.Id, &article.Title, &article.Anons, &article.ArticleText)
		if err != nil {
			panic(err)
		}

		tmpArticle = article
	}

	t.ExecuteTemplate(w, "edit", tmpArticle)
}

/*
 * Шаблон используется, если не заполнил все поля в форме создания/редактирования статьи
 */
func warning(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/warning.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}

	t.ExecuteTemplate(w, "warning", nil)
}
