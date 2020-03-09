package account

import (
	"bytes"
	"fmt"
	"html/template"
)

func style(tables []string) string {
	t, err := template.New("body").Parse(layout)
	if err != nil {
		return fmt.Sprintf("HTML Template Parse error %v", err)
	}
	body := new(bytes.Buffer)
	data := map[string]interface{}{
		"num":    len(tables),
		"tables": tables,
	}
	err = t.Execute(body, data)
	if err != nil {
		return fmt.Sprintf("HTML Template Execute error %v %+v",
			err, data)
	}
	return string(body.Bytes())
}

var layout = `
<style class="fox_global_style">
    div.fox_html_content {
        line-height: 1.5;
    }
    
    div.fox_html_content {
        font-size: 10.5pt;
        font-family: 'Microsoft YaHei UI';
        color: rgb(0, 0, 0);
        line-height: 1.5;
    }
</style>
<div><font face="微软雅黑"><br></font></div>
<div style="text-align: center;"><font face="微软雅黑"> {{ .title }} </font></div>
<div><font face="微软雅黑"><br></font></div>
<div>
    <table border="1" bordercolor="#000000" cellpadding="2" cellspacing="0" style="font-size: 10pt; border-collapse:collapse; border:none" width="50%">
        <caption><font size="2" face="微软雅黑">异常智联账号 {{ .num }} &nbsp;</font></caption>
		<tbody>
			{{ range .tables }}
            	<tr>
            	    <td width="100%" style="border: solid 1 #000000" nowrap=""><font size="2" face="微软雅黑"><div>&nbsp; {{ . }} </div></font></td>
				</tr>
			{{ end }}
        </tbody>
    </table>
</div>
`
