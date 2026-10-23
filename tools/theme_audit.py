#!/usr/bin/env python3
"""Summarize opt-in production-widget captures; no third-party Python modules."""
import argparse,collections,csv,gzip,base64,json
from pathlib import Path
from functools import lru_cache

@lru_cache(maxsize=None)
def luminance(color):
    values=[int(color[i:i+2],16)/255 for i in (1,3,5)]
    values=[v/12.92 if v<=0.04045 else ((v+0.055)/1.055)**2.4 for v in values]
    return sum(a*b for a,b in zip(values,(.2126,.7152,.0722)))
@lru_cache(maxsize=None)
def contrast(fg,bg):
    a,b=sorted((luminance(fg),luminance(bg)))
    return (b+.05)/(a+.05)

def assess(shot,bright):
    total=low=invisible=dimmed=0
    samples=[]
    for y,row in enumerate(shot['Rows']):
        line=''.join(r['Text'] for r in row).strip()
        for run in row:
            style=shot['Styles'][run['Style']]
            chars=sum(c.isalnum() for c in run['Text'])
            if style['Dim']:
                dimmed+=chars
                continue
            if not chars: continue
            fg=style['BrightFGHex' if bright else 'FGHex'];bg=style['BrightBGHex' if bright else 'BGHex']
            ratio=contrast(fg,bg)
            total+=chars
            if ratio<4.5:
                low+=chars
                if ratio<1.01: invisible+=chars
                samples.append({'ratio':round(ratio,3),'fg':fg,'bg':bg,'row':y+1,'x':run['X']+1,'text':run['Text'].strip()[:100],'line':line[:180]})
    unique={}
    for item in sorted(samples,key=lambda v:v['ratio']):
        key=(item['fg'],item['bg'],item['text'])
        unique.setdefault(key,item)
    return {'text_chars':total,'low_chars':low,'same_color_chars':invisible,'dimmed_chars':dimmed,'samples':list(unique.values())[:6]}

def signature(shot):
    return [[(r['X'],r['Text'],shot['Styles'][r['Style']]['FGHex'],shot['Styles'][r['Style']]['BGHex'],shot['Styles'][r['Style']]['Bold'],shot['Styles'][r['Style']]['Dim']) for r in row] for row in shot['Rows']]

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('raw',type=Path);parser.add_argument('output',type=Path)
    args=parser.parse_args();args.output.mkdir(parents=True,exist_ok=True)
    data=json.loads(args.raw.read_text())
    cases=[];groups=collections.defaultdict(lambda:collections.Counter())
    for shot in data['Shots']:
        for bright in (False,True):
            result=assess(shot,bright)
            row={k:shot[k] for k in ('Scene','Theme','Colors','Page','Variant','Language')}
            row.update(model='bold-bright' if bright else 'font-only',**result)
            cases.append(row)
            groups[(shot['Language'],shot['Colors'],shot['Theme'],shot['Variant'],row['model'])].update({k:result[k] for k in ('text_chars','low_chars','same_color_chars','dimmed_chars')})
    with (args.output/'lora-69-matrix.csv').open('w',encoding='utf-8-sig',newline='') as f:
        writer=csv.writer(f,lineterminator="\n");writer.writerow(['scene','theme','colors','page','variant','language','model','text_chars','low_contrast_chars','same_color_chars','dimmed_excluded_chars','examples'])
        for row in cases:
            writer.writerow([row[k] for k in ('Scene','Theme','Colors','Page','Variant','Language','model','text_chars','low_chars','same_color_chars','dimmed_chars')]+[' | '.join(f"{s['ratio']}: {s['text']} ({s['fg']}/{s['bg']})" for s in row['samples'])])
    role_rows=[]
    for role in sorted(data['Roles'],key=lambda r:(r['Colors'],r['Theme'],r['Role'])):
        style=role['Style']
        for bright in (False,True):
            fg=style['BrightFGHex' if bright else 'FGHex'];bg=style['BrightBGHex' if bright else 'BGHex']
            role_rows.append([role['Theme'],role['Colors'],role['Role'],'bold-bright' if bright else 'font-only',style['SourceFGHex'],style['SourceBGHex'],fg,bg,style['Bold'],round(contrast(fg,bg),3)])
    with (args.output/'lora-69-roles.csv').open('w',encoding='utf-8-sig',newline='') as f:
        writer=csv.writer(f,lineterminator="\n");writer.writerow(['theme','colors','role','model','source_fg','source_bg','mapped_fg','mapped_bg','bold','contrast']);writer.writerows(role_rows)
    indexed={(s['Scene'],s['Theme'],s['Colors'],s['Variant'],s['Language']):s for s in data['Shots']}
    changes=[]
    for shot in data['Shots']:
        if shot['Variant']!='fresh':continue
        other=indexed[(shot['Scene'],shot['Theme'],shot['Colors'],'roundtrip',shot['Language'])]
        if signature(shot)!=signature(other): changes.append({'scene':shot['Scene'],'theme':shot['Theme'],'colors':shot['Colors'],'language':shot['Language']})
    summary={'frames':len(data['Shots']),'scenes':len({s['Scene'] for s in data['Shots']}),'cases':cases,'role_rows':role_rows,'roundtrip_changes':changes,'totals':[dict(language=l,colors=c,theme=t,variant=v,model=m,**counts) for (l,c,t,v,m),counts in sorted(groups.items())]}
    (args.output/'lora-69-analysis.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2))
    # Compact all frames into one offline HTML file. The browser decompresses
    # the embedded data; no network, runtime server or browser storage needed.
    compact=[]
    for shot in data['Shots']:
        styles=[[s[k] for k in ('FGHex','BGHex','BrightFGHex','BrightBGHex','Bold','Dim','Underline')] for s in shot['Styles']]
        compact.append([shot['Scene'],shot['Theme'],shot['Colors'],shot['Variant'],shot['W'],shot['H'],styles,[[[r['X'],r['Text'],r['Style']] for r in row] for row in shot['Rows']],shot['Language']])
    encoded=base64.b64encode(gzip.compress(json.dumps(compact,ensure_ascii=False,separators=(',',':')).encode(),mtime=0)).decode()
    html='''<!doctype html><html lang="ko"><meta charset="utf-8"><title>LazyOra LORA-69 색상 전수 검토</title>
<style>body{font:15px system-ui,sans-serif;margin:24px;background:#f3f5f7;color:#202a36}h1{font-size:24px}p{max-width:1100px;line-height:1.6}label{margin-right:18px}select{font:inherit;padding:5px;max-width:260px}header{position:sticky;top:0;background:#f3f5f7;padding:10px 0;z-index:2}.matrix{display:grid;grid-template-columns:repeat(4,minmax(240px,1fr));gap:14px;overflow:auto}article{border:1px solid #97a4b3;background:white;border-radius:6px;padding:8px}h2{font-size:15px;margin:2px 0 8px}canvas{width:100%;height:auto;display:block;cursor:zoom-in}dialog{max-width:96vw;border:1px solid #66758a;padding:12px}dialog canvas{width:min(1280px,92vw);cursor:default}button{font:inherit;padding:5px 12px;margin:8px 0}small{display:block;margin-top:5px;color:#485869}</style>
<h1>LazyOra 색상·테마 검토 · LORA-69</h1>
<p>실제 앱 위젯 60개 화면·상태 × 12개 색상/테마 조합 × 초기/테마 왕복, 총 1,440프레임. 기본 ANSI 팔레트를 가정해 tcell의 색상 근사와 굵게=밝은색 모델을 비교합니다. 실제 SecureCRT·MobaXterm의 사용자 팔레트와 폰트는 별도 확인이 필요합니다. 그림을 누르면 확대됩니다.</p>
<header><label>언어 <select id="language"><option value="ko">한국어</option><option value="en">English</option></select></label><label>화면 <select id="scene"></select></label><label>상태 <select id="variant"><option value="fresh">초기</option><option value="roundtrip">테마 왕복 후</option></select></label><label>굵게 처리 <select id="bright"><option value="0">글꼴 굵기만</option><option value="1">기본 전경색을 밝은색으로</option></select></label></header>
<div class="matrix" id="matrix"></div><dialog id="zoom"><button id="close">닫기</button><h2></h2><canvas></canvas></dialog>
<script>
const payload='PAYLOAD';
function cellWidth(c){return /[\\u1100-\\u115f\\u2e80-\\ua4cf\\uac00-\\ud7a3\\uff01-\\uff60]/u.test(c)?2:1;}
function render(canvas,shot,bright){let [scene,theme,count,variant,w,h,styles,rows]=shot;canvas.width=w*10;canvas.height=h*24;let ctx=canvas.getContext('2d');
 for(let y=0;y<h;y++)for(let i=0;i<rows[y].length;i++){let [x,text,idx]=rows[y][i],st=styles[idx],end=i+1<rows[y].length?rows[y][i+1][0]:w;ctx.globalAlpha=1;ctx.fillStyle=st[bright?3:1];ctx.fillRect(x*10,y*24,(end-x)*10,24);}
 for(let y=0;y<h;y++)for(let [x,text,idx] of rows[y]){let st=styles[idx];ctx.fillStyle=st[bright?2:0];ctx.globalAlpha=st[5]?.5:1;ctx.font=(st[4]?'bold ':'')+'20px D2Coding,monospace';for(let ch of text){let cw=cellWidth(ch);if(ch!==' ')ctx.fillText(ch,x*10,y*24+19);if(st[6])ctx.fillRect(x*10,y*24+22,cw*10,1);x+=cw;}}
 ctx.globalAlpha=1;
}
async function start(){let bytes=Uint8Array.from(atob(payload),c=>c.charCodeAt(0));let data=JSON.parse(await new Response(new Blob([bytes]).stream().pipeThrough(new DecompressionStream('gzip'))).text());let scenes=[...new Set(data.map(s=>s[0]))];let language=document.getElementById('language'),scene=document.getElementById('scene'),variant=document.getElementById('variant'),bright=document.getElementById('bright'),matrix=document.getElementById('matrix'),zoom=document.getElementById('zoom');for(let name of scenes){let o=document.createElement('option');o.value=name;o.textContent=name;scene.append(o);}let query=new URLSearchParams(location.search);if(query.has('language'))language.value=query.get('language');if(query.has('scene'))scene.value=query.get('scene');if(query.has('bright'))bright.value=query.get('bright');if(query.has('variant'))variant.value=query.get('variant');
 function draw(){matrix.replaceChildren();for(let theme of ['dark','light','blackwhite'])for(let count of [8,16,256,16777216]){let shot=data.find(s=>s[0]===scene.value&&s[1]===theme&&s[2]===count&&s[3]===variant.value&&s[8]===language.value);let card=document.createElement('article'),title=document.createElement('h2'),canvas=document.createElement('canvas');title.textContent=theme+' · '+(count===16777216?'True Color':count+'색');card.append(title,canvas);matrix.append(card);render(canvas,shot,bright.value==='1');canvas.onclick=()=>{zoom.querySelector('h2').textContent=title.textContent+' · '+scene.value;render(zoom.querySelector('canvas'),shot,bright.value==='1');zoom.showModal();};}}
 document.getElementById('close').onclick=()=>zoom.close();for(let e of [language,scene,variant,bright])e.onchange=draw;await document.fonts.load('20px D2Coding');draw();document.body.dataset.ready='true';}
start().catch(error=>{document.body.append('불러오기 실패: '+error);});
</script></html>'''.replace('PAYLOAD',encoded).replace('60개 화면·상태',str(summary['scenes'])+'개 화면·상태').replace('1,440프레임',format(summary['frames'],',')+'프레임').replace('초기/테마 왕복,','초기/테마 왕복 × 한·영 UI,').replace('기본 ANSI 팔레트를 가정해','tcell 내장 RGB 팔레트를 기준으로')
    (args.output/'lora-69-gallery.html').write_text(html)
    print(json.dumps({'frames':summary['frames'],'scenes':summary['scenes'],'roundtrip_changes':len(changes),'gallery_bytes':len(html.encode())},ensure_ascii=False))
    print('최저 대비 역할:', sorted(role_rows,key=lambda r:r[-1])[:12])
    print('테마 왕복 차이:',changes[:12])

if __name__=='__main__':main()
