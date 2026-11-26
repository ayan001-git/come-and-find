(() => {
  const options=['Monitors','Keyboards','Mice','Webcams','Headsets','Storage'];
  const presets=[[0,1,2],[0,1,2,3],[1,2,4]];
  const words={
    ru:['Рабочее место по бюджету','Бюджет, ₸','Для чего','Учёба','Видеозвонки','Игры','Что включить','Подобрать комплект','Выберите минимум две категории.','Ни один продавец не предлагает весь комплект в этом бюджете.','Остаток бюджета','Весь комплект в корзину','Комплект добавлен','В корзине товары другого продавца.','Обновите подбор: цена или наличие могли измениться.','Минимальная цена полного комплекта'],
    kk:['Бюджетке сай жұмыс орны','Бюджет, ₸','Мақсат','Оқу','Бейнеқоңырау','Ойын','Қажетті жабдық','Жинақ таңдау','Кемінде екі санатты таңдаңыз.','Бұл бюджетке толық жинақ ұсынатын сатушы табылмады.','Қалған қаражат','Жинақты себетке қосу','Жинақ себетке қосылды','Себетте басқа сатушының тауары бар.','Баға немесе қор өзгеруі мүмкін. Қайта іздеңіз.','Толық жинақтың ең төмен бағасы'],
    en:['Build a workspace','Budget, ₸','Use','Study','Video calls','Gaming','Equipment','Find bundles','Select at least two categories.','No seller has the complete bundle within this budget.','Budget remaining','Add bundle to cart','Bundle added','Your cart contains another seller’s items.','Refresh the selection. Price or stock may have changed.','Lowest complete bundle price']
  };
  const word=i=>words[window.cfLang][i];
  const form=$('#kitForm'), results=$('#kitResults');
  let revision=0, chosen=new Set(presets[0]), preset=0;
  function clear(){revision++;results.replaceChildren()}
  function labels(){
    clear();
    $('#kitTitle').textContent=word(0); $('#kitBudgetLabel').textContent=word(1);
    $('#kitPresetLabel').textContent=word(2); $('#kitSearch').textContent=word(7);
    $('#kitPreset').replaceChildren(...presets.map((_,i)=>new Option(word(i+3),i)));
    $('#kitPreset').value=preset;
    $('#kitCategories').innerHTML='<legend>'+word(6)+'</legend>'+options.map((key,i)=>
      '<label><input type="checkbox" value="'+i+'" '+(chosen.has(i)?'checked':'')+'>'+esc(categoryName(key))+'</label>').join('');
  }
  form.oninput=clear;
  $('#kitPreset').onchange=e=>{preset=Number(e.target.value);chosen=new Set(presets[preset]);labels()};
  $('#kitCategories').onchange=e=>{const i=Number(e.target.value);if(e.target.checked)chosen.add(i);else chosen.delete(i)};
  form.onsubmit=async e=>{
    e.preventDefault();clear();const serial=revision;
    if(chosen.size<2){results.textContent=word(8);return}
    const budget=Number($('#kitBudget').value)*100;
    try {
      const products=await api('/api/products');
      if(serial!==revision)return;
      const sellers=new Map();
      for(const p of products){
        if(!p.stock || !chosen.has(options.indexOf(p.category)))continue;
        if(!sellers.has(p.supplier_id))sellers.set(p.supplier_id,new Map());
        const group=sellers.get(p.supplier_id),old=group.get(p.category);
        if(!old||p.price_kzt<old.price_kzt)group.set(p.category,p);
      }
      const bundles=[...sellers.values()].filter(g=>g.size===chosen.size).map(g=>{
        const items=[...g.values()];
        return {items,total:items.reduce((sum,p)=>sum+Math.round(p.price_kzt*100),0)};
      }).sort((a,b)=>a.total-b.total);
      for(const {items,total} of bundles.filter(b=>b.total<=budget)){
        const card=document.createElement('article');card.className='panel';
        card.innerHTML='<h3>'+esc(items[0].supplier)+'</h3><ul>'+items.map(p=>'<li>'+esc(localizeProduct(p).name)+' · '+fmt(p.price_kzt)+'</li>').join('')+'</ul><strong>'+fmt(total/100)+'</strong><p>'+word(10)+' · '+fmt((budget-total)/100)+'</p>';
        const button=document.createElement('button');button.className='btn';button.textContent=word(11);
        button.onclick=async()=>{
          if(!me){openAuth();return}
          if(me.role!=='buyer'){toast(t('cart.onlyBuyer'));return}
          button.disabled=true;
          try{await api('/api/cart/bundle',{method:'POST',body:JSON.stringify({ids:items.map(p=>p.id),total_tiyn:total})});await refreshCartCount();button.textContent=word(12)}
          catch(err){toast(err.message==='Different cart supplier'?word(13):word(14));button.disabled=false}
        };
        card.append(button);results.append(card);
      }
      if(!results.childElementCount)results.textContent=word(9)+(bundles.length?' '+word(15)+' · '+fmt(bundles[0].total/100):'');
    }catch(err){if(serial===revision)results.textContent=t('toast.searchError')}
  };
  window.addEventListener('cf-language-change',labels);labels();
})();