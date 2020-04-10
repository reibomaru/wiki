package main

import (
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http" //HTTPプロトコルを利用してくれるパッケージ
	"os"
	"regexp" //正規表現のパッケージ

	_ "github.com/go-sql-driver/mysql"
)

//Page wikiのデータ構造
type Page struct {
	Title string //タイトル
	Body  []byte //タイトルの中身
}

type Wiki struct {
	ID      int
	Title   string
	Content string
}

//パスのアドレスを設定して文字の長さを定数として持つ
const lenPath = len("/view/")

//テンプレートファイルの配列を作成,string型のキーにtemplate.Template型の要素を持つ
var templates = make(map[string]*template.Template)

//正規表現でURLを生成できる大文字小文字の英字と数字を判別する
//^は1文字目のチェック []その中の文字の種類 $最後の文字にマッチ +1文字以上
var titleValidator = regexp.MustCompile("^[a-zA-Z0-9]+$")

//.txt
const expendString = ".txt"

//初期化関数
func init() {
	for _, tmpl := range []string{"edit", "view"} {
		//エラーの場合Panicを起こすためエラー処理はなし
		t := template.Must(template.ParseFiles("template/" + tmpl + ".html"))
		templates[tmpl] = t
	}
}

//タイトルのチェックを行う
func getTitle(w http.ResponseWriter, r *http.Request) (title string, err error) {
	title = r.URL.Path[lenPath:]
	if !titleValidator.MatchString(title) {
		http.NotFound(w, r)
		err = errors.New("Invalid Page Title")
		log.Print(err)
	}
	return
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) { //*http.RequestではURLの情報にアクセスできる
	p, err := loadPage(title)
	if err != nil {
		//editHandlerのURLに飛ばすことで編集ページに飛ばすことができます。
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}

	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title) //
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

//
func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	log.Print(r.FormValue("body"))
	body := r.FormValue("body")

	db, err := sql.Open("mysql", "root:Reibo1998@@/go_wiki")
	if err != nil {
		panic(err.Error())
	}

	stmtInsert, err := db.Prepare("INSERT INTO wikis(title, content) VALUES(?,?)")
	if err != nil {
		panic(err.Error())
	}

	_, err = stmtInsert.Exec(title, body)
	if err != nil {
		panic(err.Error())
	}

	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func topHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", "root:Reibo1998@@/go_wiki")
	if err != nil {
		panic(err.Error())
	}

	rows, err := db.Query("SELECT title FROM wikis")
	if err != nil {
		panic(err.Error())
	}
	var wiki Wiki
	var titles []string
	for rows.Next() {
		err := rows.Scan(&wiki.Title)
		if err != nil {
			panic(err.Error())
		}
		titles = append(titles, wiki.Title)
	}
	t := template.Must(template.ParseFiles("template/top.html"))
	err = t.Execute(w, titles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//Requestからページタイトルを取り出して、fnを呼び出す
		title := r.URL.Path[lenPath:]
		if !titleValidator.MatchString(title) {
			http.NotFound(w, r)
			err := errors.New("Invalid Page Title")
			log.Print(err)
			return
		}
		fn(w, r, title)
	}
}

//画面に要素を出力する関数
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	//htmlの中にTitleやBodyを入れれるようにする
	err := templates[tmpl].Execute(w, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//テキストファイルの保存メソッド
// func (p *Page) save() error {
// 	//タイトルの名前でテキストファイルを作成して保存します。
// 	filename := p.Title + ".txt"

// 	//0600は、テキストデータを書き込んだり読み込んだりする権限を設定しています。
// 	return ioutil.WriteFile(filename, p.Body, 0600)
// }

//titleからファイル名を読み込んで新しいPageのポインタを返す
func loadPage(title string) (*Page, error) {
	// filename := title + ".txt"
	// body, err := ioutil.ReadFile(filename)
	// //errに値が入ったらエラーとしてbodyの値をnilにして返す
	// if err != nil {
	// 	return nil, err
	// }
	db, err := sql.Open("mysql", "root:Reibo1998@@/go_wiki")
	if err != nil {
		panic(err.Error())
	}

	rows, err := db.Query("select title, content from wikis where title = ?", title)
	if err != nil {
		return nil, err
	}

	var wiki Wiki
	for rows.Next() {
		err = rows.Scan(&wiki.Title, &wiki.Content)
		// log.Print(wiki.Title)
	}
	return &Page{Title: wiki.Title, Body: []byte(wiki.Content)}, nil //Pageをポインタで返す
}

func main() {
	dir, _ := os.Getwd()
	http.Handle("/new/", http.StripPrefix("/new/", http.FileServer(http.Dir(dir+"/static"))))
	http.HandleFunc("/view/", makeHandler(viewHandler)) //パスを指定してどういった動きにするのかをハンドリングする
	http.HandleFunc("/edit/", makeHandler(editHandler)) //パスを指定してどういった動きにするのかをハンドリングする
	http.HandleFunc("/save/", makeHandler(saveHandler)) //パスを指定してどういった動きにするのかをハンドリングする
	http.HandleFunc("/top/", topHandler)                //パスを指定してどういった動きにするのかをハンドリングする
	http.ListenAndServe(":3000", nil)                   //サーバーを自分のPCの中で立ち上げている、ポートを8080としてたちあげる
}
