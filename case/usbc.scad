//usbc();
difference() {
    union(){
        cylinder(1.5, 10, 10);
        cylinder(6, 8, 8);
    }

translate([0,3,0])
    linear_extrude(0.3, center=true) mirror([1,0,0])
        text("Si-", size=4,halign="center");

translate([0,-7,0])
    linear_extrude(0.3, center=true) mirror([1,0,0])
        text("Gnal", size=4,halign="center");

    rotate([90,0,0]) usbc();
}

module usbc() {
    translate([0,10+1,0])
    union(){
        cube([9.2, 30, 3.5], center=true);
        translate([0,5,-3.3/2]) cube([12.2, 30, 1.2], center=true);
    }
    
}