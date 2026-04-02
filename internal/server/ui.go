package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Surveyor</title>
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#c45d2c;--rl:#e8753a;--leather:#a0845c;--ll:#c4a87a;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c44040;--mono:'JetBrains Mono',Consolas,monospace;--serif:'Libre Baskerville',Georgia,serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);font-size:13px;line-height:1.6}
a{color:var(--rl);text-decoration:none}a:hover{color:var(--gold)}
.hdr{padding:.7rem 1.2rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}
.hdr h1{font-family:var(--serif);font-size:1rem}.hdr h1 span{color:var(--rl)}
.stats{display:flex;gap:1.5rem;font-size:.7rem;color:var(--leather)}.stats b{color:var(--cream);font-weight:600}
.main{max-width:800px;margin:0 auto;padding:1.5rem}
.section-label{font-size:.65rem;letter-spacing:2px;text-transform:uppercase;color:var(--rust);margin:2rem 0 .8rem}
.btn{font-family:var(--mono);font-size:.75rem;padding:.4rem .8rem;border:1px solid;cursor:pointer;background:transparent;transition:all .15s}
.btn-p{border-color:var(--rust);color:var(--rl)}.btn-p:hover{background:var(--rust);color:var(--cream)}
.btn-g{border-color:var(--gold);color:var(--gold)}.btn-g:hover{background:var(--gold);color:var(--bg)}
.btn-d{border-color:var(--bg3);color:var(--cm)}.btn-d:hover{border-color:var(--red);color:var(--red)}
.btn-s{border-color:var(--green);color:var(--green)}.btn-s:hover{background:var(--green);color:var(--bg)}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:1rem;margin-bottom:.8rem}
.card h3{font-size:.85rem;margin-bottom:.3rem}.card p{font-size:.75rem;color:var(--cd)}
.card-meta{font-size:.65rem;color:var(--leather);margin:.3rem 0}.card-actions{display:flex;gap:.4rem;margin-top:.5rem}
input[type=text],input[type=url],textarea,select{background:var(--bg);border:1px solid var(--bg3);color:var(--cream);padding:.4rem .6rem;font-family:var(--mono);font-size:.8rem;width:100%;outline:none;margin-bottom:.5rem}
input:focus,textarea:focus,select:focus{border-color:var(--rust)}
textarea{resize:vertical;min-height:60px}
.field-row{display:flex;gap:.5rem;align-items:center;margin-bottom:.4rem;padding:.3rem .5rem;background:var(--bg);border:1px solid var(--bg3)}
.field-row input,.field-row select{margin:0;width:auto;flex:1}
.field-row .btn{padding:.2rem .4rem;font-size:.65rem}
.modal-bg{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.6);display:flex;align-items:center;justify-content:center;z-index:100}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:90%;max-width:600px;max-height:85vh;overflow-y:auto}
.modal h2{font-size:.9rem;margin-bottom:1rem}
.resp-table{width:100%;border-collapse:collapse;font-size:.7rem;margin-top:.5rem}
.resp-table th{background:var(--bg3);padding:.3rem .5rem;text-align:left;color:var(--ll);font-size:.6rem;text-transform:uppercase;letter-spacing:.5px}
.resp-table td{padding:.3rem .5rem;border-bottom:1px solid var(--bg3);color:var(--cd);max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.empty{text-align:center;padding:2rem;color:var(--cm);font-style:italic;font-family:var(--serif)}
</style>
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital@0;1&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
</head><body>
<div class="hdr"><h1><span>Surveyor</span></h1><div class="stats">Forms: <b id="sf">-</b> &nbsp; Responses: <b id="sr">-</b></div></div>
<div class="main">
<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem">
<div class="section-label" style="margin:0">Your forms</div>
<button class="btn btn-p" onclick="showCreate()">+ New Form</button>
</div>
<div id="forms"></div>
</div>
<div id="modal"></div>
<script>
let forms=[];
async function load(){
  const r=await fetch('/api/forms');const d=await r.json();forms=d.forms||[];
  const s=await fetch('/api/stats');const st=await s.json();
  document.getElementById('sf').textContent=st.forms;
  document.getElementById('sr').textContent=st.responses;
  render();
}
function render(){
  const el=document.getElementById('forms');
  if(!forms.length){el.innerHTML='<div class="empty">No forms yet. Create one to get started.</div>';return}
  el.innerHTML=forms.map(f=>'<div class="card"><h3>'+esc(f.title)+'</h3>'+(f.description?'<p>'+esc(f.description)+'</p>':'')+
    '<div class="card-meta">'+f.fields.length+' fields · '+f.response_count+' responses · '+(f.active?'<span style="color:var(--green)">active</span>':'<span style="color:var(--cm)">closed</span>')+' · <a href="/f/'+f.slug+'" target="_blank">/f/'+f.slug+'</a></div>'+
    '<div class="card-actions"><button class="btn btn-g" onclick="viewResponses(\''+f.id+'\')">Responses</button><button class="btn btn-p" onclick="editForm(\''+f.id+'\')">Edit</button><button class="btn btn-s" onclick="copyLink(\''+f.slug+'\')">Copy link</button><button class="btn btn-d" onclick="exportCSV(\''+f.id+'\')">CSV</button><button class="btn btn-d" onclick="delForm(\''+f.id+'\',\''+esc(f.title)+'\')">Delete</button></div></div>').join('')
}
function esc(s){return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function showCreate(){showFormEditor(null)}
function editForm(id){const f=forms.find(x=>x.id===id);if(f)showFormEditor(f)}
function showFormEditor(f){
  const isNew=!f;
  f=f||{title:'',description:'',slug:'',webhook_url:'',fields:[],active:true};
  let fieldsHTML=f.fields.map((fd,i)=>fieldRow(fd,i)).join('');
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>'+(isNew?'Create form':'Edit form')+'</h2>'+
    '<label style="font-size:.7rem;color:var(--leather)">Title</label><input type="text" id="f-title" value="'+esc(f.title)+'">'+
    '<label style="font-size:.7rem;color:var(--leather)">Slug (URL path)</label><input type="text" id="f-slug" value="'+esc(f.slug)+'" placeholder="my-form">'+
    '<label style="font-size:.7rem;color:var(--leather)">Description</label><textarea id="f-desc" rows="2">'+esc(f.description)+'</textarea>'+
    '<label style="font-size:.7rem;color:var(--leather)">Webhook URL (optional)</label><input type="url" id="f-wh" value="'+esc(f.webhook_url)+'" placeholder="https://...">'+
    '<label style="font-size:.7rem;color:var(--leather)">Active</label><select id="f-active"><option value="1"'+(f.active?' selected':'')+'>Yes</option><option value="0"'+(!f.active?' selected':'')+'>No</option></select>'+
    '<div style="display:flex;justify-content:space-between;align-items:center;margin:1rem 0 .5rem"><label style="font-size:.7rem;color:var(--leather);margin:0">Fields</label><button class="btn btn-p" style="font-size:.6rem;padding:.15rem .4rem" onclick="addField()">+ Field</button></div>'+
    '<div id="fields">'+fieldsHTML+'</div>'+
    '<div style="display:flex;gap:.5rem;margin-top:1rem"><button class="btn btn-p" onclick="saveForm('+(isNew?'null':'"'+f.id+'"')+')">'+(isNew?'Create':'Save')+'</button><button class="btn btn-d" onclick="closeModal()">Cancel</button></div></div></div>'
}
let fieldIdx=0;
function fieldRow(fd,i){
  fd=fd||{name:'',label:'',type:'text',required:false,options:[]};
  const id='fd-'+(fieldIdx++);
  return'<div class="field-row" id="'+id+'"><input type="text" placeholder="name" value="'+esc(fd.name)+'" style="max-width:100px"><input type="text" placeholder="Label" value="'+esc(fd.label)+'"><select><option value="text"'+(fd.type==='text'?' selected':'')+'>text</option><option value="email"'+(fd.type==='email'?' selected':'')+'>email</option><option value="number"'+(fd.type==='number'?' selected':'')+'>number</option><option value="textarea"'+(fd.type==='textarea'?' selected':'')+'>textarea</option><option value="select"'+(fd.type==='select'?' selected':'')+'>select</option><option value="radio"'+(fd.type==='radio'?' selected':'')+'>radio</option><option value="checkbox"'+(fd.type==='checkbox'?' selected':'')+'>checkbox</option></select><label style="font-size:.6rem;white-space:nowrap"><input type="checkbox"'+(fd.required?' checked':'')+'> req</label><button class="btn btn-d" onclick="this.parentElement.remove()">x</button></div>'
}
function addField(){document.getElementById('fields').insertAdjacentHTML('beforeend',fieldRow(null,0))}
function collectFields(){
  const rows=document.querySelectorAll('#fields .field-row');
  return Array.from(rows).map(r=>{
    const inputs=r.querySelectorAll('input[type=text]');
    const sel=r.querySelector('select');
    const chk=r.querySelector('input[type=checkbox]');
    return{name:inputs[0].value,label:inputs[1].value,type:sel.value,required:chk.checked,options:[]}
  }).filter(f=>f.name)
}
async function saveForm(id){
  const body={title:document.getElementById('f-title').value,slug:document.getElementById('f-slug').value,description:document.getElementById('f-desc').value,webhook_url:document.getElementById('f-wh').value,active:document.getElementById('f-active').value==='1',fields:collectFields()};
  const method=id?'PUT':'POST';const url=id?'/api/forms/'+id:'/api/forms';
  const r=await fetch(url,{method,headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
  if(!r.ok){const d=await r.json();alert(d.error);return}
  closeModal();load()
}
async function delForm(id,name){if(!confirm('Delete form "'+name+'"?'))return;await fetch('/api/forms/'+id,{method:'DELETE'});load()}
async function viewResponses(id){
  const f=forms.find(x=>x.id===id);
  const r=await fetch('/api/forms/'+id+'/responses');const d=await r.json();
  const resps=d.responses||[];
  const fieldNames=f.fields.map(f=>f.name);
  let table='<table class="resp-table"><thead><tr><th>Time</th>'+fieldNames.map(n=>'<th>'+esc(n)+'</th>').join('')+'</tr></thead><tbody>';
  if(!resps.length){table+='<tr><td colspan="'+(fieldNames.length+1)+'" style="text-align:center;font-style:italic;color:var(--cm)">No responses yet</td></tr>'}
  resps.forEach(resp=>{
    const t=new Date(resp.created_at).toLocaleString();
    table+='<tr><td>'+t+'</td>'+fieldNames.map(n=>'<td>'+(resp.data[n]||'')+'</td>').join('')+'</tr>'
  });
  table+='</tbody></table>';
  document.getElementById('modal').innerHTML='<div class="modal-bg" onclick="if(event.target===this)closeModal()"><div class="modal"><h2>Responses: '+esc(f.title)+' ('+d.total+')</h2>'+table+'<div style="margin-top:1rem;display:flex;gap:.5rem"><button class="btn btn-d" onclick="exportCSV(\''+id+'\')">Export CSV</button><button class="btn btn-d" onclick="closeModal()">Close</button></div></div></div>'
}
function closeModal(){document.getElementById('modal').innerHTML=''}
function copyLink(slug){navigator.clipboard.writeText(window.location.origin+'/f/'+slug).then(()=>alert('Copied: '+window.location.origin+'/f/'+slug))}
function exportCSV(id){window.open('/api/forms/'+id+'/responses/export','_blank')}
load();setInterval(load,30000)
</script></body></html>`
