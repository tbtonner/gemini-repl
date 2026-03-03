package themes

var Kanagawa = []byte(`{
	"document": {
		"block_prefix": "\n",
		"block_suffix": "\n",
		"margin": 2,
		"color": "#DCD7BA"
	},
	"heading": {
		"color": "#C0A36E",
		"bold": true,
		"block_prefix": "\n",
		"block_suffix": ""
	},
	"h1": {
		"color": "#938AA9",
		"suffix": " ",
		"underline": true,
		"bold": true
	},
	"block_quote": {
		"color": "#727169",
		"indent": 1,
		"indent_token": "▎",
		"italic": true
	},
	"list": {
		"level_indent": 2,
		"color": "#FFA066"
	},
	"code": {
		"color": "#98BB6C",
		"background_color": "#16161D",
		"italic": false
	},
	"code_block": {
		"margin": 2,
		"color": "#DCD7BA",
		"background_color": "#16161D",
		"chroma": {
			"text": { "color": "#DCD7BA" },
			"keyword": { "color": "#957FB8" },
			"keyword_namespace": { "color": "#957FB8" },
			"keyword_type": { "color": "#7AA89F" },
			"string": { "color": "#98BB6C" },
			"comment": { "color": "#727169", "italic": true },
			"name": { "color": "#DCD7BA" },
			"name_function": { "color": "#7E9CD8" },
			"name_variable": { "color": "#E6C384" },
			"name_type": { "color": "#7AA89F" },
			"literal_number": { "color": "#D27E99" },
			"operator": { "color": "#C0A36E" },
			"punctuation": { "color": "#DCD7BA" }
		}
	},
	"table": {
		"center_separator": "┼",
		"column_separator": "│",
		"row_separator": "─",
		"color": "#727169"
	},
	"strong": {
		"color": "#FFA066",
		"bold": true
	},
	"emphasis": {
		"color": "#C8C093",
		"italic": true
	},
	"link": {
		"color": "#7E9CD8",
		"underline": true
	},
	"link_text": {
		"color": "#7FB4CA",
		"bold": true
	}
}`)
