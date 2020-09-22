// TODO Replace go asset embedding once in 1.16
package smtpclient

const Subject = `Participa en {{.OrgName}} con Vocdoni`

const TextTemplate = `
Versió en castellà

Hola {{.Name}},

{{.OrgName}} te ha invitado a unirte a su plataforma de gobernanza.

Como miembro de {{.OrgName}}, recibirás todas las noticias de la organización en tu móvil y podrás participar en la toma de decisiones mediante voto digital seguro.

Para unirte a {{.OrgName}}:
1. Instala Vocdoni para Android o iPhone desde la tienda de aplicaciones de tu dispositivo.
2. Crea tu identidad dentro de la app, copia el siguiente enlace en tu navegador y sigue los pasos:

{{.ValidationLink}}

{{if .OrgEmail}}Para más información o ayuda, contacta con {{.OrgEmail}}.{{end}}
Gracias,

{{.OrgName}}

Versió en català

Hola {{.Name}},

{{.OrgName}} t'ha convidat a unir-te a la seva plataforma de governança.

Com a membre de {{.OrgName}}, rebràs totes les notícies de l'organització en el teu mòbil i podràs participar en la presa de decisions mitjançant vot digital segur.

Per unir-te a {{.OrgName}}:
1. Instal·la Vocdoni per a Android o iPhone des de la botiga d'aplicacions del teu dispositiu.
2. Crea la teva identitat dins de l'app, còpia el següent enllaç en el navegador i segueix el passos:

{{.ValidationLink}}

{{if .OrgEmail}}Per a més informació o ajuda, contacta amb {{.OrgEmail}}.{{end}}
Gràcies,

{{.OrgName}}
`

const HTMLTemplate = `
<!-- FILE: ../manager-backend/misc/mail/template_catesp.mjml -->
<!doctype html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office">

<head>
  <title>
  </title>
  <!--[if !mso]><!-- -->
  <meta http-equiv="X-UA-Compatible" content="IE=edge">
  <!--<![endif]-->
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style type="text/css">
    #outlook a {
      padding: 0;
    }

    body {
      margin: 0;
      padding: 0;
      -webkit-text-size-adjust: 100%;
      -ms-text-size-adjust: 100%;
    }

    table,
    td {
      border-collapse: collapse;
      mso-table-lspace: 0pt;
      mso-table-rspace: 0pt;
    }

    img {
      border: 0;
      height: auto;
      line-height: 100%;
      outline: none;
      text-decoration: none;
      -ms-interpolation-mode: bicubic;
    }

    p {
      display: block;
      margin: 13px 0;
    }
  </style>
  <!--[if mso]>
        <xml>
        <o:OfficeDocumentSettings>
          <o:AllowPNG/>
          <o:PixelsPerInch>96</o:PixelsPerInch>
        </o:OfficeDocumentSettings>
        </xml>
        <![endif]-->
  <!--[if lte mso 11]>
        <style type="text/css">
          .mj-outlook-group-fix { width:100% !important; }
        </style>
        <![endif]-->
  <!--[if !mso]><!-->
  <link href="https://fonts.googleapis.com/css?family=Open+Sans:300,400,500,700" rel="stylesheet" type="text/css">
  <link href="https://fonts.googleapis.com/css?family=Ubuntu:300,400,500,700" rel="stylesheet" type="text/css">
  <style type="text/css">
    @import url(https://fonts.googleapis.com/css?family=Open+Sans:300,400,500,700);
    @import url(https://fonts.googleapis.com/css?family=Ubuntu:300,400,500,700);
  </style>
  <!--<![endif]-->
  <style type="text/css">
    @media only screen and (min-width:480px) {
      .mj-column-per-100 {
        width: 100% !important;
        max-width: 100%;
      }
    }
  </style>
  <style type="text/css">
    @media only screen and (max-width:480px) {
      table.mj-full-width-mobile {
        width: 100% !important;
      }

      td.mj-full-width-mobile {
        width: auto !important;
      }
    }
  </style>
</head>

<body>
  <div style="">
    <!--[if mso | IE]>
      <table
         align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600"
      >
        <tr>
          <td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;">
      <![endif]-->
    <div style="margin:0px auto;max-width:600px;">
      <table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;">
        <tbody>
          <tr>
            <td style="direction:ltr;font-size:0px;padding:20px 0;padding-bottom:0px;padding-top:20px;text-align:center;">
              <!--[if mso | IE]>
                  <table role="presentation" border="0" cellpadding="0" cellspacing="0">
                
        <tr>
      
            <td
               class="" style="vertical-align:top;width:600px;"
            >
          <![endif]-->
              <div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;">
                <table border="0" cellpadding="0" cellspacing="0" role="presentation" width="100%">
                  <tbody>
                    <tr>
                      <td style="background-color:#f7f7f7;vertical-align:top;padding:20px;">
                        <table border="0" cellpadding="0" cellspacing="0" role="presentation" style="" width="100%">
                          <tr>
                            <td align="center" style="font-size:0px;padding:10px 25px;padding-top:0;padding-right:0px;padding-bottom:30px;padding-left:0px;word-break:break-word;">
                              <table border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-collapse:collapse;border-spacing:0px;">
                                <tbody>
                                  <tr>
                                    <td style="width:140px;">
                                      <img alt="" height="auto" src="logoVoc.png" style="border:none;display:block;outline:none;text-decoration:none;height:auto;width:100%;font-size:13px;" width="140" />
                                    </td>
                                  </tr>
                                </tbody>
                              </table>
                            </td>
                          </tr>
                          <tr>
                            <td align="center" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:14px;font-weight:700;line-height:1;text-align:center;color:#000000;">Versión en castellano</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Hola {{.Name}},</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">{{.OrgName}} te ha invitado a unirte a su plataforma de gobernanza.</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Como miembro de {{.OrgName}}, recibirás todas las noticias de la organización en tu móvil y podrás participar en la toma de decisiones mediante voto digital seguro.</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Para unirte a {{.OrgName}}:</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">1. Instala Vocdoni para <a href="https://play.google.com/store/apps/details?id=org.vocdoni.app">Android</a> o <a href="https://apps.apple.com/es/app/vocdoni/id1505234624">iPhone</a></div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">2. Crea tu cuenta en la app y <a href="{{.ValidationLink}}">sigue este link</a> en tu móvil.</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Si tienes problemas con el enlace, copia el siguiente texto en el navegador:</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;text-align:left;color:#000000;">
                                <pre><small>{{.ValidationLink}}</small></pre>
                              </div>
                            </td>
                          </tr>
                          <tr>
                            <td style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <p style="border-top:solid 1px #cccccc;font-size:1px;margin:0px auto;width:100%;">
                              </p>
                              <!--[if mso | IE]>
        <table
           align="center" border="0" cellpadding="0" cellspacing="0" style="border-top:solid 1px #cccccc;font-size:1px;margin:0px auto;width:510px;" role="presentation" width="510px"
        >
          <tr>
            <td style="height:0;line-height:0;">
              &nbsp;
            </td>
          </tr>
        </table>
      <![endif]-->
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">{{if .OrgEmail}}Para más información o ayuda, <a href="mailto:{{.OrgEmail}}">contacta con {{.OrgName}}</a>.{{end}}</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Gracias,<br /></br />{{.OrgName}}</div>
                            </td>
                          </tr>
                        </table>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <!--[if mso | IE]>
            </td>
          
        </tr>
      
                  </table>
                <![endif]-->
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <!--[if mso | IE]>
          </td>
        </tr>
      </table>
      
      <table
         align="center" border="0" cellpadding="0" cellspacing="0" class="" style="width:600px;" width="600"
      >
        <tr>
          <td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;">
      <![endif]-->
    <div style="margin:0px auto;max-width:600px;">
      <table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:100%;">
        <tbody>
          <tr>
            <td style="direction:ltr;font-size:0px;padding:20px 0;padding-bottom:0px;padding-top:20px;text-align:center;">
              <!--[if mso | IE]>
                  <table role="presentation" border="0" cellpadding="0" cellspacing="0">
                
        <tr>
      
            <td
               class="" style="vertical-align:top;width:600px;"
            >
          <![endif]-->
              <div class="mj-column-per-100 mj-outlook-group-fix" style="font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;">
                <table border="0" cellpadding="0" cellspacing="0" role="presentation" width="100%">
                  <tbody>
                    <tr>
                      <td style="background-color:#f7f7f7;vertical-align:top;padding:20px;">
                        <table border="0" cellpadding="0" cellspacing="0" role="presentation" style="" width="100%">
                          <tr>
                            <td align="center" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:14px;font-weight:700;line-height:1;text-align:center;color:#000000;">Versió en català</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Hola {{.Name}},</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">{{.OrgName}} t'ha convidat a unir-te a la seva plataforma de governança.</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Com a membre de {{.OrgName}}, rebràs totes les notícies de l'organització en el teu mòbil i podràs participar en la presa de decisions mitjançant vot digital segur.</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Per unir-te a {{.OrgName}}:</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">1. Instal·la Vocdoni per a <a href="https://play.google.com/store/apps/details?id=org.vocdoni.app">Android</a> o <a href="https://apps.apple.com/es/app/vocdoni/id1505234624">iPhone</a></div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">2. Crea la teva identitat dins de l'app i <a href="{{.ValidationLink}}">segueix aquest link</a> en el teu mòbil.</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Si tens problemes amb l'enllaç, còpia el següent text en el navegador:</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;text-align:left;color:#000000;">
                                <pre><small>{{.ValidationLink}}</small></pre>
                              </div>
                            </td>
                          </tr>
                          <tr>
                            <td style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <p style="border-top:solid 1px #cccccc;font-size:1px;margin:0px auto;width:100%;">
                              </p>
                              <!--[if mso | IE]>
        <table
           align="center" border="0" cellpadding="0" cellspacing="0" style="border-top:solid 1px #cccccc;font-size:1px;margin:0px auto;width:510px;" role="presentation" width="510px"
        >
          <tr>
            <td style="height:0;line-height:0;">
              &nbsp;
            </td>
          </tr>
        </table>
      <![endif]-->
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">{{if .OrgEmail}}Per a més informació o ajuda, <a href="mailto:{{.OrgEmail}}">contacta amb {{.OrgName}}</a>.{{end}}</div>
                            </td>
                          </tr>
                          <tr>
                            <td align="left" style="font-size:0px;padding:10px 25px;word-break:break-word;">
                              <div style="font-family:Open Sans, sans-serif;font-size:15px;line-height:1;text-align:left;color:#000000;">Gràcies,<br /></br />{{.OrgName}}</div>
                            </td>
                          </tr>
                        </table>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <!--[if mso | IE]>
            </td>
          
        </tr>
      
                  </table>
                <![endif]-->
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <!--[if mso | IE]>
          </td>
        </tr>
      </table>
      <![endif]-->
  </div>
</body>

</html>
`
const LogoVocBase64 = `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
RK5CYII=`
