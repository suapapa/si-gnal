
difference() {
    union() {
        // phone
        spacer(0,0, 5, 5);
        spacer(109,18, 5, 3);
        // rpi 3b
        spacer(25,10-5, 5, 2.5);
        spacer(25,10+58-5, 5, 2.5);
        spacer(25+49,10-5, 5, 2.5);
        spacer(25+49,10+58-5, 5, 2.5);
        //amp
        spacer(25+49+20, -10+45, 5, 2);
        spacer(25+49+20+12.5, -10+45, 5, 2);
        
        translate([-7,-5,0]) cube([109+10,10+58-10,1.5]);
        translate([25-5,5,0]) cube([49+10, 58+5, 1.5]);
        
        translate([5-1,5-1+10,0]) cube([90+2,10+2,4]);
        translate([5-1,40-1,0]) cube([90+2,20+2,4]);    
    }

    //phone
    thruhole(0,0, 2);
    translate([0,0,1.5]) cylinder(40, 6.5/2+0.2, 6.5/2+0.2, $fn=6);
    thruhole(43,33, 1.5);
    thruhole(109,18, 1.5);
    // rpi 3b
    thruhole(25,10-5, 1.2);
    thruhole(25+49,10-5, 1.2);
    thruhole(25,10+58-5, 1.2);
    thruhole(25+49,10+58-5, 1.2);
    //amp
    thruhole(25+49+20, -10+45, 1);
    thruhole(25+49+20+12.5, -10+45, 1);
    
    translate([5,5+10,-2]) cube([90,10,14]);
    translate([5,40,-2]) cube([90,20,14]);    
}

difference() {
    spacer(0, -10, 7, 3);
    thruhole(0, -10, 1.5);
}

difference() {
    spacer(10, -12, 4, 4);
    thruhole(10, -12, 2);
}

module thruhole(x, y, r) {
    translate([x, y, 0]) cylinder(40, r, r, center=true);
}

module spacer(x, y, h, r) {
    translate([x, y, 0]) cylinder(h, r, r, $fn=6);
}