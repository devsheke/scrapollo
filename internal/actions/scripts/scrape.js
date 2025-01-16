() => {
  let leads = [];
  const rows = document.querySelectorAll('.zp_tFLCQ .zp_hWv1I');

  for (let i = 0; i < rows.length; i++) {
    const columns = rows[i].querySelectorAll('.zp_KtrQp');
    let lead = {
      name: columns[1].innerText.replaceAll('\n------', ''),
      title: columns[2].innerText,
      company: columns[3].innerText,
      location: columns[8].innerText,
      employees: columns[9].innerText,
      industry: columns[10].innerText.replaceAll('\n', ','),
      keywords: columns[11].innerText.replaceAll('\n', ','),
    };

    let links = [];
    const linksColumn = columns[7].querySelectorAll('a');
    for (const link of linksColumn) {
      if (link.href !== '') links.push(link.href);
    }
    lead.links = links.join(',');

    const emailSpan = columns[4].querySelector('.zp_xvo3G');
    if (emailSpan !== null) {
      lead.email = columns[4].querySelector('.zp_xvo3G').innerText;
      lead.phone = columns[5].innerText;
      leads.push(lead);
      continue;
    }

    const emailButton = columns[4].querySelector('button');
    if (emailButton === null) {
      continue;
    }

    emailButton.click();
    let retries = 0;
    while (retries < 30) {
      const emailSpan = columns[4].querySelector('.zp_xvo3G');
      if (emailSpan === null) {
        new Promise((resolve) => setTimeout(resolve, 2000)).then((_) => { });
        retries++;
      } else {
        lead.email = columns[4].querySelector('.zp_xvo3G').innerText;
        lead.phone = columns[5].innerText;
        break;
      }
    }

    leads.push(lead);
  }

  return leads;
};
