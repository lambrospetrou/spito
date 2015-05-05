// options: {
//				seconds: #,
//				intervalUpdate: 1000
//				onUpdate: function(remaining){}
//				onEnd: function(){}
//			}
//
function Countdown(options) {
	this.seconds = options.seconds || 10;
	this.intervalUpdate = options.intervalUpdate || 1000;
	this.onUpdate = options.onUpdate || function(rem){}
	this.onEnd = options.onEnd || function(){}
	this.timer = null;
}

Countdown.prototype.start = function() {
	clearInterval(this.timer);
	
	// save the THIS object since inside the decrement function
	// this will represent that function
	var _this = this;

	var decrement = function () {
		--_this.seconds;
		_this.onUpdate(_this.seconds);
		if (0 === _this.seconds) {
			_this.stop();
			_this.onEnd();
		}
	}
	// notify me each second
	this.timer = setInterval(decrement, this.intervalUpdate);
}

Countdown.prototype.stop = function () {
	clearInterval(this.timer);
	this.timer = null;
}