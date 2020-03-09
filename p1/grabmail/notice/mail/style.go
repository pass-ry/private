package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"time"
)

func style(from, to time.Time, tables map[string]*table) string {
	t, err := template.New("body").Parse(layout)
	if err != nil {
		return fmt.Sprintf("HTML Template Parse error %v", err)
	}
	body := new(bytes.Buffer)
	data := map[string]interface{}{
		"title": fmt.Sprintf("统计时间：%s 至 %s",
			from.Format("2006-01-02 15:04:05"),
			to.Format("2006-01-02 15:04:05")),
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

<div style="text-align: left; font-size: 16px;">
	<span style="line-height: 22px;">
		<font face="微软雅黑">{{ .title }}</font>
	</span>
</div>

<div style="text-align: left;"><font face="微软雅黑"><br></font></div>

{{ range $name, $table := .tables }}
	<div style="text-align: left;"><font face="微软雅黑"><br></font></div>
	<table border="1" bordercolor="#000000" cellpadding="2" cellspacing="0" style="font-size: 10pt; border-collapse:collapse; border:none" width="50%">
		<caption style="font-size: 16px;">
			<font size="2" style="font-size: 16px;" face="微软雅黑">
				<b>{{ $name }}</b>
			</font>
		</caption>
		<tbody>
			<tr>
				{{ range $table.Columns }}
					<td width="20%" style="border: solid 1 #000000" nowrap="">
						<font size="2" face="微软雅黑">
							<div style="text-align: center;">
								<b>{{ . }}&nbsp;</b>
							</div>
						</font>
					</td>
				{{ end }}
			</tr>
			{{ range $row := $table.Rows }}
				<tr>
					{{ range $row }}
						<td width="20%" style="border: solid 1 #000000" nowrap="">
							<font size="2" face="微软雅黑">
								<div>&nbsp;{{ . }}</div>
							</font>
						</td>
					{{ end }}
				</tr>
			{{ end }}
		</tbody>
	</table>
	<div style="text-align: left;"><font face="微软雅黑"><br></font></div>
{{ end }}

<font face="微软雅黑"><br></font>
<div><font face="微软雅黑"><br></font></div>
<div><font face="微软雅黑"><br></font></div>
`
