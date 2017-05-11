package main

var defaultPageHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=no">
  <title>{{.Title}} - Powered by Tower</title>
    <style>
      *{font-family: Helvetica Neue, Arial, Verdana, sans-serif;}
      body{margin: 0;}
      .header{
        width:100%;
        height: 70px;
        background-color: burlywood;
      }
      h1{
        font-size: 30px;
        line-height: 70px;
        max-width: 880px;
        margin: 0 auto;
        padding-left: 20px;
      }
      .content{
        max-width: 880px;
        margin: 0 auto;
        padding-left:20px;
      }
      h2{font-size:20px;}
      .message{margin: 40px 0 60px 0;}
      .snippet, .trace{
        margin-left: -15px;
        padding:14px;
        border: 1px solid burlywood;
        border-radius: 5px;
        -moz-border-radius: 5px;
        -webkit-border-radius: 5px;
        margin-bottom: 30px;
      }
      dl{margin:0}
      .numbers, .codes{line-height: 22px;}
      .bold{font-weight:bold;}
      dd.codes{margin:0;min-height:22px}
      .numbers{
        float:left;
        text-align: right;
        margin-right: 15px;
        color: #929292;
      }
      .trace ul{
        padding: 0;
        margin: 0;
        list-style: none;
      }
      .trace ul li{margin-bottom: 10px;}
      .trace .func{color: #929292;}
      .clearfix{clear: both;}
    </style>
  </head>
  <body>
    <div class="header">
      <h1>{{.Title}} -- {{.Time}}</h1>
    </div>

    <div class="content">
      <div class="message">
        {{.Message}}
      </div>

      {{if .ShowSnippet}}
      <h2>{{.SnippetPath}}</h2>
      <div class="snippet">
          {{range .Snippet}}
          <dl>
            {{if .Current}}
              <dt class="numbers bold">{{.Number}}</dt>
              <dd class="codes bold">{{.Code}}</dd>
            {{else}}
              <dt class="numbers">{{.Number}}</dt>
              <dd class="codes">{{.Code}}</dd>
            {{end}}
          </dl>
          {{end}}
      </div>
      {{end}}


      {{if .ShowTrace}}
      <h2>Trace</h2>
      <div class="trace">
        <ul>
          {{range .Trace}}
          <li>
            {{if .AppFile}}
              <strong>{{.File}}</strong>
            {{end}}
            {{if not .AppFile}}
              {{.File}}
            {{end}}
            <br/>
            <span class="func">{{.Func}}</span>
          </li>
          {{end}}
        </ul>
      </div>
      {{end}}
    </div>
  </body>
</html>
`
