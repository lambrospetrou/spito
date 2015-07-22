/**
 * Created by lambros on 04/05/15.
 */
$(function(){
    var ink, d, x, y;
    $(".ripplelink").click(function(e){
        var ink = $(this).find('.ink');
        if (ink.length === 0) {
            $(this).prepend("<span class='ink'></span>");
            ink = $(this).find('.ink');
        }
        ink.removeClass("animate");

        if(!ink.height() && !ink.width()){
            d = Math.max($(this).outerWidth(), $(this).outerHeight());
            ink.css({height: d, width: d});
        }

        x = e.pageX - $(this).offset().left - ink.width()/2;
        y = e.pageY - $(this).offset().top - ink.height()/2;

        ink.css({top: y+'px', left: x+'px'}).addClass("animate");
    });
});
