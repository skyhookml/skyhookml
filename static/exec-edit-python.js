Vue.component('exec-edit-python', {
	data: function() {
		return {
			code: '',
		};
	},
	props: ['node'],
	created: function() {
		this.code = this.node.Params;
	},
	methods: {
		autoindent: function(e) {
			var el = e.target;
			if(e.keyCode == 9) {
				// tab
				e.preventDefault();
				var start = el.selectionStart;
				var prefix = this.code.substring(0, start);
				var suffix = this.code.substring(el.selectionEnd);
				this.code = prefix + '\t' + suffix;

				Vue.nextTick(function() {
					el.selectionStart = start+1;
					el.selectionEnd = start+1;
				});
			} else if(e.keyCode == 13) {
				// enter
				e.preventDefault();
				var start = el.selectionStart;
				var prefix = this.code.substring(0, start);
				var suffix = this.code.substring(el.selectionEnd);
				var prevLine = prefix.lastIndexOf('\n');

				var spacing = '';
				if(prevLine != -1) {
					for(var i = prevLine+1; i < start; i++) {
						var char = this.code[i];
						if(char != '\t' && char != ' ') {
							break;
						}
						spacing += char;
					}
				}
				this.code = prefix + '\n' + spacing + suffix;
				Vue.nextTick(function() {
					el.selectionStart = start+1+spacing.length;
					el.selectionEnd = el.selectionStart;
				});
			}
		},
		save: function() {
			myCall('POST', '/exec-nodes/'+this.node.ID, JSON.stringify({
				Params: this.code,
			}));
		},
	},
	template: `
<div class="flex-container">
	<div class="flex-container">
		<textarea v-model="code" v-on:keydown="autoindent($event)" class="el-big" placeholder="Your Code Here"></textarea>
	</div>
	<div class="m-1">
		<button v-on:click="save" type="button" class="btn btn-primary btn-sm el-wide">Save</button>
	</div>
</div>
	`,
});
