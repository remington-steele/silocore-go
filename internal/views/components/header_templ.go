// Code generated by templ - DO NOT EDIT.

// templ: version: v0.3.833
package components

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

func Header() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		templ_7745c5c3_Err = templruntime.WriteString(templ_7745c5c3_Buffer, 1, "<header class=\"bg-white shadow\"><div class=\"container mx-auto px-4 py-4\"><div class=\"flex justify-between items-center\"><div class=\"flex items-center\"><a href=\"/\" class=\"text-xl font-bold text-primary-600\">SiloCore</a></div><nav class=\"hidden md:flex space-x-6\"><a href=\"/orders\" class=\"text-gray-600 hover:text-primary-600 transition-colors\">Orders</a> <a href=\"/profile\" class=\"text-gray-600 hover:text-primary-600 transition-colors\">Profile</a><div class=\"relative\" x-data=\"{ open: false }\"><button class=\"flex items-center text-gray-600 hover:text-primary-600 transition-colors focus:outline-none\" hx-get=\"/api/tenant/switch\" hx-target=\"#tenant-dropdown\" hx-trigger=\"click\" hx-swap=\"innerHTML\"><span>Tenant</span> <svg class=\"ml-1 w-4 h-4\" fill=\"none\" stroke=\"currentColor\" viewBox=\"0 0 24 24\" xmlns=\"http://www.w3.org/2000/svg\"><path stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M19 9l-7 7-7-7\"></path></svg></button><div id=\"tenant-dropdown\" class=\"absolute right-0 mt-2 w-48 bg-white rounded-md shadow-lg py-1 z-10 hidden\"><!-- Tenant list will be loaded here via HTMX --></div></div></nav><div class=\"flex items-center\"><form hx-post=\"/logout\" hx-confirm=\"Are you sure you want to log out?\"><button type=\"submit\" class=\"text-gray-600 hover:text-primary-600 transition-colors\">Logout</button></form></div><button class=\"md:hidden focus:outline-none\" hx-get=\"/api/menu/mobile\" hx-target=\"#mobile-menu\" hx-trigger=\"click\" hx-swap=\"innerHTML\"><svg class=\"w-6 h-6 text-gray-600\" fill=\"none\" stroke=\"currentColor\" viewBox=\"0 0 24 24\" xmlns=\"http://www.w3.org/2000/svg\"><path stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\" d=\"M4 6h16M4 12h16M4 18h16\"></path></svg></button></div><div id=\"mobile-menu\" class=\"md:hidden mt-4 hidden\"><!-- Mobile menu will be loaded here via HTMX --></div></div></header>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return nil
	})
}

var _ = templruntime.GeneratedTemplate
